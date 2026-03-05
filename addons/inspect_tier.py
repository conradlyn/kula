#!/usr/bin/env python3
"""
Utility to inspect Kula tier files.
"""

import struct
import sys
import os
import datetime
import json
from typing import Optional, Tuple, BinaryIO

MAGIC = b"KULASPIE"
HEADER_SIZE = 64


def parse_header(buf: bytes) -> Tuple[bytes, int, int, int, int, int, int]:
    """Unpack fixed-size header, excluding reserved bytes."""
    # Unpack the 64-byte header:
    # 0:8   magic (8 bytes string)
    # 8:16  version (uint64)
    # 16:24 max data size (uint64)
    # 24:32 write offset (uint64)
    # 32:40 total records written (uint64)
    # 40:48 oldest timestamp (int64, unix nano)
    # 48:56 newest timestamp (int64, unix nano)
    # 56:64 reserved (8 bytes)
    unpacked = struct.unpack("<8sQQQQqq8s", buf)
    return unpacked[:7]  # Return all except reserved


def find_latest_record(
    f: BinaryIO, wrapped: bool, write_off: int, max_data: int
) -> Optional[bytes]:
    """Locate the most recently written record in the ring buffer."""
    segments = []
    if wrapped:
        segments.append((write_off, max_data - write_off))
        segments.append((0, write_off))
    else:
        segments.append((0, write_off))

    last_data = None
    for start, size in segments:
        f.seek(HEADER_SIZE + start)
        bytes_read = 0
        while bytes_read < size:
            if size - bytes_read < 4:
                break

            len_buf = f.read(4)
            if len(len_buf) < 4:
                break

            data_len = struct.unpack("<I", len_buf)[0]
            if data_len == 0 or data_len > max_data:
                break

            record_len = 4 + data_len
            if bytes_read + record_len > size:
                break

            data = f.read(data_len)
            if len(data) < data_len:
                break

            last_data = data
            bytes_read += record_len
    return last_data


def print_record(last_data: bytes) -> None:
    """Attempt to parse and print the record as JSON."""
    try:
        parsed = json.loads(last_data.decode("utf-8"))
        print("\nLatest Record:")
        print(json.dumps(parsed, indent=2))
    except (json.JSONDecodeError, UnicodeDecodeError):
        print(f"\nLatest Record (failed to parse JSON): {last_data!r}")


def inspect_tier(filepath: str) -> None:
    """Reads and displays information from a Kula tier file."""
    try:
        file_size = os.path.getsize(filepath)
        with open(filepath, "rb") as f:
            buf = f.read(HEADER_SIZE)
            if len(buf) < HEADER_SIZE:
                print(
                    f"Error: File too small ({len(buf)} bytes, expected {HEADER_SIZE} bytes)",
                    file=sys.stderr,
                )
                sys.exit(1)

            (
                magic,
                version,
                max_data,
                write_off,
                count,
                oldest_nano,
                newest_nano,
            ) = parse_header(buf)

            if magic != MAGIC:
                magic_repr = magic.decode("utf-8", errors="replace")
                print(f"Error: Invalid magic: {magic_repr}", file=sys.stderr)
                sys.exit(1)

            wrapped = (
                write_off > 0 and count > 0 and file_size >= HEADER_SIZE + max_data
            )
            print(f"File: {filepath}")
            print(f"Version: {version}")

            current_data = max_data if wrapped else write_off
            pct = (current_data / max_data * 100) if max_data > 0 else 0.0
            print(f"Data Size: {current_data} / {max_data} bytes ({pct:.2f}%)")
            print(f"Write Offset: {write_off}")
            print(f"Total Records: {count}")

            # Timestamps
            oldest_ts = (
                datetime.datetime.fromtimestamp(oldest_nano / 1e9).astimezone()
                if oldest_nano > 0
                else None
            )
            newest_ts = (
                datetime.datetime.fromtimestamp(newest_nano / 1e9).astimezone()
                if newest_nano > 0
                else None
            )

            print(
                f"Oldest Timestamp: {oldest_ts.isoformat() if oldest_ts else '(none)'}"
            )
            print(
                f"Newest Timestamp: {newest_ts.isoformat() if newest_ts else '(none)'}"
            )
            print(f"Wrapped: {wrapped}")

            if oldest_ts and newest_ts:
                print(f"Time Range Covered: {newest_ts - oldest_ts}")

            if count == 0:
                print("\nLatest Record: (none)")
                return

            last_data = find_latest_record(f, wrapped, write_off, max_data)
            if last_data:
                print_record(last_data)
            else:
                print("\nLatest Record: (none found)")

    except OSError as err:
        print(f"Error inspecting tier file: {err}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python inspect_tier.py <path-to-tier-file>", file=sys.stderr)
        sys.exit(1)

    inspect_tier(sys.argv[1])
