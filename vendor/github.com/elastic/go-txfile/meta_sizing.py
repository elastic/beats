#!/usr/bin/env python

import argparse
import pprint


list_header = 8 + 4  # next pointer + page entry count


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('-p', '--pagesize', dest='pagesize', type=long, default=4096)
    parser.add_argument('-s', '--maxsize', dest='maxsize', type=long, default=1 << 30)
    parser.add_argument('-w', '--wal', dest='wal', type=long, default=1000)
    args = parser.parse_args()

    stats = compute_stats(args.pagesize, args.maxsize, args.wal)
    pprint.pprint(stats, indent=2)


def compute_stats(page_size, max_size, wal_entries):
    # multiply by 2, as next transaction might require same amount
    # of pages

    max_pages = max_size / page_size

    stats = {
        "pagesize": page_size,
        "max_size": max_size,
        "max_pages": max_pages,
        "wal_entries": wal_entries,
    }

    wal_meta = wal_mapping_pages(page_size, wal_entries)
    stats['wal_meta'] = 2 * wal_meta
    stats['wal_meta_bytes'] = 2 * wal_meta * page_size
    stats['wal_meta_bytes_io_per_tx'] = wal_meta * page_size

    freelist = freelist_pages(page_size, max_pages)
    stats['freelist_pages'] = 2 * freelist
    stats['freelist_bytes'] = 2 * freelist * page_size
    stats['freelist_bytes_io_per_tx'] = freelist * page_size

    file_header = 2
    stats['file header'] = file_header

    count = wal_meta + wal_entries + 2 * freelist + file_header
    stats['min_meta_pages'] = count

    # meta allocator grows in power of 2
    meta_pages = next_power_of_2(count)
    internal_frag = meta_pages - count
    data_pages = max_pages - meta_pages

    stats['meta_pages'] = meta_pages
    stats['data_pages'] = data_pages
    stats['meta_bytes'] = meta_pages * page_size
    stats['data_bytes'] = data_pages * page_size
    stats['internal_fragmentation'] = internal_frag
    stats['meta_percentage'] = 100.0 * float(meta_pages) / float(max_pages)
    stats['data_percentage'] = 100.0 * float(data_pages) / float(max_pages)
    stats['frag_percentage'] = 100.0 * float(internal_frag) / float(max_pages)

    return stats


def pages(entries, entries_per_page):
    return (entries + (entries_per_page - 1)) / entries_per_page


def freelist_pages(page_size, max_pages):
    """Compute max number of freelist pages required.
       Assumes full fragmentation, such that every second page is free.
       Due to run-length-encoding of freelist entries, this assumption gets us
       the max number of freelist entries."""

    # estimate of max number of free pages with full fragmentation
    entries = (max_pages + 1) / 2

    avail = page_size - list_header
    entries_per_page = avail / 8  # 8 byte per entry

    return pages(entries, entries_per_page)


def wal_mapping_pages(page_size, entries):
    """Compute number of required pages for the wal id mapping"""
    entries_per_page = (page_size - list_header) / 14  # 14 byte per entry
    return pages(entries, entries_per_page)


def next_power_of_2(x):
    return 1 << (x-1).bit_length()


if __name__ == "__main__":
    main()
