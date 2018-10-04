#!/usr/bin/env perl

use strict;
use warnings;
use feature qw( say );

use MaxMind::DB::Writer::Tree;

my $filename = $ARGV[0];

# See https://metacpan.org/pod/MaxMind::DB::Writer::Tree#DATA-TYPES
my %types = (
    city                => 'map',
    continent           => 'map',
    country             => 'map',
    registered_country  => 'map',
    subdivisions        => ['array', 'map'],

    code                => 'utf8_string',
    iso_code            => 'utf8_string',
    geoname_id          => 'uint32',

    names               => 'map',
    en                  => 'utf8_string',

    location            => 'map',
    latitude            => 'double',
    longitude           => 'double',
    time_zone           => 'utf8_string',
);

my $tree = MaxMind::DB::Writer::Tree->new(
    database_type => 'GeoIP2-City',
    description => { en => 'GeoIP test Fixtures' },
    ip_version => 6,

    # add a callback to validate data going in to the database
    map_key_type_callback => sub { $types{ $_[0] } },

    # "record_size" is the record size in bits.  Either 24, 28 or 32.
    record_size => 24,
);

my $middle_earth = {
    names => {
        en => 'Middle Earth',
    },
    code  => 'ME',
};

my $arnor = {
    names => {
        en => 'Kingdom of Arnor',
    },
    iso_code => 'AR',
};

my $gondor = {
    names => {
        en => 'Kingdom of Gondor',
    },
    iso_code => 'GO',
};

my $shire = {
    names => {
        en => 'The Shire',
    },
    iso_code => 'SH',
};

my $pelennor = {
    names => {
        en => 'Pelennor',
    },
    iso_code => 'PE',
};

my $hobbiton = {
    continent => $middle_earth,
    country => $arnor,
    subdivisions => [$shire],
    city => {
        names => {
            en => 'Hobbiton',
        },
    },
    location => {
        latitude => 52.4908,
        longitude => 13.3275,
    },
};

my $minas_tirith = {
    continent => $middle_earth,
    country => $gondor,
    subdivisions => [$pelennor],
    city => {
        names => {
            en => 'Minas Tirith',
        },
    },
    location => {
        latitude => 40.7143,
        longitude => -74.0060,
    },
};

my %networks = (
    '85.181.35.0/24' => $hobbiton,
    '1.2.0.0/16' => $minas_tirith,
    '199.96.0.0/16' => $minas_tirith,
    '116.31.0.0/16' => $minas_tirith,

    '2a03:0000:10ff:f00f:0000:0000:0:8000/64' => $minas_tirith,
);

for my $network ( keys %networks ) {
    $tree->insert_network( $network, $networks{$network} );
}

# Write the database to disk.
open my $fh, '>:raw', $filename;
$tree->write_tree( $fh );
close $fh;

say "$filename has now been created";
