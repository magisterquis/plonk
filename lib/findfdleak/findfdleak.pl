#!/usr/bin/env perl
# 
# findfdleak.pl
# Finds file descriptor leaks
# By J. Stuart McMurray
# Created 20231213
# Last Modified 20231213

use warnings;
use strict;

# Watch for opened and closed files.
my %fdn;       # file descriptor -> name
my $open = 0;  # In an open call
my $name = ""; # File name
my $fd   = ""; # File descriptor number
while (<>) {
        if (/CALL\s+close\((\d+)\)$/) { 
                # Closed files are easy.
                if (exists $fdn{$1}) {
                        delete $fdn{$1};
                }
                $open = 0;
        } elsif (/CALL\s+open(at)?\(/) {
                # If we're opening a file, we'll need to get it, as well as its
                # name and the returned file descriptor number.
                $open = 1;
                $name = $fd = "";
        } elsif ($open and /RET\s+open(at)?\s(\d+)/) {
                # Got the fd
                if ("" ne $name) {
                        $fdn{$2} = $name;
                } else {
                        $fd = $2;
                }
        } elsif ($open and /NAMI\s+\"(.*)\"$/) {
                # Got the name.
                if ("" ne $fd) {
                        $fdn{$fd} = $1;
                } else {
                        $name = $1;
                }
        } else {
                # Something else entirely.
                $open = 0;
        }
}

# Figure out which files we have left.
for my $fd (sort {$a <=> $b} keys %fdn) {
        print "$fd\t$fdn{$fd}\n";
}
