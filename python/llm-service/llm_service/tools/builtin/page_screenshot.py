"""
Page Screenshot Tool - Capture a screenshot of any URL and upload to S3.

Uses Playwright service for capture and shared screenshot_utils for S3 upload.
Designed for competitor monitoring visual evidence.
"""

import asyncio
import os
import base64
import logging
from typing import Dict, List, Optional, Any
from urllib.parse import urlparse

import aiohttp

from ..base import Tool, ToolMetadata, ToolParameter, ToolParameterType, ToolResult
from .screenshot_utils import upload_screenshot_to_s3

logger = logging.getLogger(__name__)

PLAYWRIGHT_SERVICE_URL = os.getenv("PLAYWRIGHT_SERVICE_URL", "http://playwright-service:8002")


class PageScreenshotTool(Tool):
    """Capture a full-page screenshot of a URL and return a permanent S3 URL."""

    def __init__(self):
        super().__init__()

    def _get_metadata(self) -> ToolMetadata:
        return ToolMetadata(
            name="page_screenshot",
            version="1.0.0",
            description="Capture a screenshot of a web page and return a permanent URL",
            category="retrieval",
            author="Shannon",
            requires_auth=False,
            rate_limit=10,
            timeout_seconds=45,
            memory_limit_mb=256,
            sandboxed=True,
            dangerous=False,
            cost_per_use=0.0,
        )

    def _get_parameters(self) -> List[ToolParameter]:
        return [
            ToolParameter(
                name="url",
                type=ToolParameterType.STRING,
                description="The URL of the page to screenshot",
                required=True,
            ),
            ToolParameter(
                name="full_page",
                type=ToolParameterType.BOOLEAN,
                description="Whether to capture the full scrollable page (default: true)",
                required=False,
                default=True,
            ),
            ToolParameter(
                name="include_text",
                type=ToolParameterType.BOOLEAN,
                description="Also extract visible text content from the page (for content hashing)",
                required=False,
                default=False,
            ),
        ]

    async def _execute_impl(
        self, session_context: Optional[Dict] = None, **kwargs
    ) -> ToolResult:
        url = kwargs.get("url", "")
        full_page = kwargs.get("full_page", True)
        include_text = kwargs.get("include_text", False)

        if not url:
            logger.warning("page_screenshot: missing url parameter")
            return ToolResult(success=False, output=None, error="url parameter is required")

        # Validate URL scheme to prevent SSRF (file://, data:, internal IPs via non-http)
        parsed = urlparse(url)
        if parsed.scheme not in ("http", "https"):
            logger.warning(f"page_screenshot: unsupported scheme {parsed.scheme!r} for {url}")
            return ToolResult(success=False, output=None, error=f"Only http/https URLs are supported, got: {parsed.scheme!r}")

        if not PLAYWRIGHT_SERVICE_URL:
            logger.warning("page_screenshot: PLAYWRIGHT_SERVICE_URL not configured")
            return ToolResult(success=False, output=None, error="PLAYWRIGHT_SERVICE_URL not configured")

        # 1. Capture screenshot via Playwright service
        try:
            async with aiohttp.ClientSession() as session:
                payload = {
                    "url": url,
                    "full_page": full_page,
                    "wait_ms": 3000,
                    "include_text": include_text,
                }
                timeout = aiohttp.ClientTimeout(total=45)
                async with session.post(
                    f"{PLAYWRIGHT_SERVICE_URL}/capture",
                    json=payload,
                    timeout=timeout,
                ) as response:
                    if response.status != 200:
                        error_text = await response.text()
                        error_msg = f"Playwright service error: {response.status} - {error_text[:200]}"
                        logger.warning(f"page_screenshot failed for {url}: {error_msg}")
                        return ToolResult(
                            success=False,
                            output=None,
                            error=error_msg,
                        )

                    result = await response.json()

                    if not result.get("success"):
                        error_msg = result.get("error", "Screenshot capture failed")
                        logger.warning(f"page_screenshot failed for {url}: {error_msg}")
                        return ToolResult(
                            success=False,
                            output=None,
                            error=error_msg,
                        )

                    screenshot_b64 = result.get("screenshot")
                    if not screenshot_b64:
                        logger.warning(f"page_screenshot failed for {url}: no screenshot data in Playwright response")
                        return ToolResult(
                            success=False,
                            output=None,
                            error="No screenshot data returned from Playwright",
                        )

                    try:
                        screenshot_bytes = base64.b64decode(screenshot_b64)
                    except (ValueError, Exception) as e:
                        logger.warning(f"page_screenshot failed for {url}: invalid base64: {e}")
                        return ToolResult(
                            success=False,
                            output=None,
                            error=f"Invalid base64 screenshot data: {e}",
                        )

        except (aiohttp.ClientError, asyncio.TimeoutError) as e:
            logger.warning(f"page_screenshot failed for {url}: Playwright connection error: {e}")
            return ToolResult(success=False, output=None, error=f"Playwright connection error: {e}")

        # 2. Upload to S3
        session_id = ""
        if session_context:
            session_id = session_context.get("session_id", "")

        screenshot_url = await upload_screenshot_to_s3(
            image_bytes=screenshot_bytes,
            url=url,
            label="page",
            media_type="image/png",
            session_id=session_id,
        )

        if not screenshot_url:
            logger.warning(f"page_screenshot failed for {url}: S3 upload failed or not configured")
            return ToolResult(
                success=False,
                output=None,
                error="S3 upload failed or S3 not configured (S3_LP_SCREENSHOT_BUCKET env var required)",
            )

        logger.info(f"page_screenshot: captured {url} -> {screenshot_url} ({len(screenshot_bytes)/1024:.1f}KB)")

        output = {
            "screenshot_url": screenshot_url,
            "url": url,
            "size_bytes": len(screenshot_bytes),
            "title": result.get("title", ""),
        }
        if include_text:
            output["text_content"] = result.get("text_content", "")

        return ToolResult(
            success=True,
            output=output,
            cost_usd=0.001,
            cost_model="shannon_page_screenshot",
            tokens_used=500,
        )
