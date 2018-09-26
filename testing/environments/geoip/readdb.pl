#!/usr/bin/env perl
 
use strict;
use warnings;
use feature qw( say );
 
use Data::Printer;
use MaxMind::DB::Reader;
 
my $db = shift @ARGV or die 'Usage: perl readdb.pl [db] [ip_address]';
my $ip = shift @ARGV or die 'Usage: perl readdb.pl [db] [ip_address]';
 
my $reader = MaxMind::DB::Reader->new( file => $db );
 
say 'Description: ' . $reader->metadata->{description}->{en};
 
my $record = $reader->record_for_address( $ip );
say np $record;
