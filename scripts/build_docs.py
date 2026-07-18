#!/usr/bin/env python3
"""Build the Indonesian and English documentation sites."""

from __future__ import annotations

import argparse
import shutil
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SOURCE_DIR = ROOT / "docs"
STAGING_DIR = ROOT / ".docs-build"
SITE_DIR = ROOT / "site"

PUBLIC_DOCS = (
    "index",
    "getting-started",
    "query-builder",
    "create-update",
    "scan",
    "pagination",
    "dry-run",
    "transactions",
    "dialects",
    "examples",
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--no-strict",
        action="store_true",
        help="Allow MkDocs warnings without failing the build.",
    )
    return parser.parse_args()


def stage_documentation() -> None:
    if STAGING_DIR.exists():
        shutil.rmtree(STAGING_DIR)

    for language in ("id", "en"):
        (STAGING_DIR / language).mkdir(parents=True)

    for document in PUBLIC_DOCS:
        english_source = SOURCE_DIR / f"{document}.md"
        indonesian_source = SOURCE_DIR / f"{document}.id.md"

        for source in (english_source, indonesian_source):
            if not source.is_file():
                raise FileNotFoundError(f"Missing documentation source: {source}")

        shutil.copy2(english_source, STAGING_DIR / "en" / f"{document}.md")

        indonesian_content = indonesian_source.read_text(encoding="utf-8")
        indonesian_content = indonesian_content.replace(".id.md)", ".md)")
        (STAGING_DIR / "id" / f"{document}.md").write_text(
            indonesian_content,
            encoding="utf-8",
        )


def prepare_output_directory() -> None:
    if SITE_DIR.exists():
        shutil.rmtree(SITE_DIR)
    SITE_DIR.mkdir(parents=True)


def build_language(config: str, strict: bool) -> None:
    command = [sys.executable, "-m", "mkdocs", "build", "--clean", "-f", config]
    if strict:
        command.append("--strict")
    subprocess.run(command, cwd=ROOT, check=True)


def write_root_redirect() -> None:
    redirect = """<!doctype html>
<html lang="id">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta http-equiv="refresh" content="0; url=./id/">
    <link rel="canonical" href="./id/">
    <title>Dokumentasi ORM</title>
    <script>window.location.replace("./id/" + window.location.search + window.location.hash);</script>
  </head>
  <body>
    <p><a href="./id/">Buka dokumentasi Bahasa Indonesia</a></p>
  </body>
</html>
"""
    SITE_DIR.mkdir(parents=True, exist_ok=True)
    (SITE_DIR / "index.html").write_text(redirect, encoding="utf-8")
    (SITE_DIR / ".nojekyll").touch()


def main() -> None:
    args = parse_args()
    stage_documentation()
    prepare_output_directory()
    build_language("mkdocs.id.yml", strict=not args.no_strict)
    build_language("mkdocs.en.yml", strict=not args.no_strict)
    write_root_redirect()


if __name__ == "__main__":
    main()
