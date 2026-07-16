#!/usr/bin/env python3
"""Upload markdown documents with slug frontmatter to the RxLab docs service."""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from pathlib import Path
from typing import Iterator
from urllib.error import HTTPError, URLError
from urllib.parse import quote
from urllib.request import Request, urlopen


DEFAULT_ENDPOINT = "https://autopilot.rxlab.app"
BATCH_SIZE = 50
FRONTMATTER_DELIMITER = "---"
KEY_PATTERN = re.compile(r"^[A-Za-z_][A-Za-z0-9_-]*$")


class DocsUploadError(Exception):
    """Raised when documents cannot be collected or uploaded."""


def parse_frontmatter(path: Path) -> tuple[dict[str, str], str]:
    text = path.read_text(encoding="utf-8-sig")
    lines = text.splitlines(keepends=True)
    if not lines or lines[0].strip() != FRONTMATTER_DELIMITER:
        return {}, text

    closing_index = next(
        (
            index
            for index, line in enumerate(lines[1:], start=1)
            if line.strip() == FRONTMATTER_DELIMITER
        ),
        None,
    )
    if closing_index is None:
        raise DocsUploadError(f"{path}: frontmatter is missing a closing ---")

    metadata: dict[str, str] = {}
    for line_number, line in enumerate(lines[1:closing_index], start=2):
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        if ":" not in stripped:
            raise DocsUploadError(
                f"{path}:{line_number}: expected a 'key: value' frontmatter entry"
            )
        key, value = stripped.split(":", 1)
        key = key.strip()
        value = value.strip()
        if not KEY_PATTERN.fullmatch(key):
            raise DocsUploadError(f"{path}:{line_number}: invalid frontmatter key {key!r}")
        if len(value) >= 2 and value[0] == value[-1] and value[0] in {'"', "'"}:
            value = value[1:-1]
        metadata[key] = value

    body = "".join(lines[closing_index + 1 :]).lstrip("\r\n")
    return metadata, body


def collect_documents(docs_dir: Path) -> list[dict[str, str]]:
    if not docs_dir.is_dir():
        raise DocsUploadError(f"docs directory does not exist: {docs_dir}")

    documents: list[dict[str, str]] = []
    slug_paths: dict[str, Path] = {}
    for path in sorted(docs_dir.rglob("*.md")):
        metadata, body = parse_frontmatter(path)
        slug = metadata.get("slug", "").strip()
        if not slug:
            continue
        if slug in slug_paths:
            raise DocsUploadError(
                f"duplicate slug {slug!r}: {slug_paths[slug]} and {path}"
            )
        slug_paths[slug] = path
        documents.append({"docId": slug, "content": body})

    if not documents:
        raise DocsUploadError(
            f"no markdown documents with slug frontmatter found in {docs_dir}"
        )
    return documents


def batches(
    items: list[dict[str, str]], size: int = BATCH_SIZE
) -> Iterator[list[dict[str, str]]]:
    for start in range(0, len(items), size):
        yield items[start : start + size]


def upload_batch(url: str, token: str, documents: list[dict[str, str]]) -> str:
    payload = json.dumps({"documents": documents}).encode("utf-8")
    request = Request(
        url,
        data=payload,
        method="POST",
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
            "Accept": "application/json",
            "User-Agent": "rxlab-docs-uploader/1.0",
        },
    )

    try:
        with urlopen(request, timeout=60) as response:
            response_body = response.read().decode("utf-8", errors="replace")
            status = response.status
    except HTTPError as error:
        response_body = error.read().decode("utf-8", errors="replace")
        raise DocsUploadError(
            f"docs service returned HTTP {error.code}: {response_body or error.reason}"
        ) from error
    except URLError as error:
        raise DocsUploadError(f"could not reach docs service: {error.reason}") from error

    if not 200 <= status < 300:
        raise DocsUploadError(f"docs service returned HTTP {status}: {response_body}")

    if not response_body:
        return "accepted"
    try:
        response_json = json.loads(response_body)
    except json.JSONDecodeError:
        return response_body
    return str(response_json.get("jobId") or response_json.get("message") or "accepted")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="validate and display batches without making network requests",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    repo_root = Path(__file__).resolve().parent.parent
    documents = collect_documents(repo_root / "docs")
    document_batches = list(batches(documents))

    if args.dry_run:
        print(
            f"Dry run: {len(documents)} documents in {len(document_batches)} "
            f"batch{'es' if len(document_batches) != 1 else ''}."
        )
        for index, batch in enumerate(document_batches, start=1):
            slugs = ", ".join(document["docId"] for document in batch)
            print(f"Batch {index} ({len(batch)}): {slugs}")
        return 0

    endpoint = os.environ.get("DOCS_ENDPOINT", DEFAULT_ENDPOINT).strip().rstrip("/")
    repository = os.environ.get("DOCS_REPOSITORY_ID", "").strip()
    token = os.environ.get("DOCS_UPLOAD_TOKEN", "").strip()
    if not repository:
        raise DocsUploadError("DOCS_REPOSITORY_ID is required")
    if not token:
        raise DocsUploadError("DOCS_UPLOAD_TOKEN is required")

    repository_path = quote(repository, safe="")
    url = f"{endpoint}/api/v1/docs/repositories/{repository_path}/documents"
    for index, batch in enumerate(document_batches, start=1):
        result = upload_batch(url, token, batch)
        print(
            f"Uploaded batch {index}/{len(document_batches)} "
            f"({len(batch)} documents): {result}"
        )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (DocsUploadError, OSError) as error:
        print(f"error: {error}", file=sys.stderr)
        raise SystemExit(1)
