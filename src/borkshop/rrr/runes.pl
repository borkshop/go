# Hello friends.
#
# This is a Perl script that reads runes.txt on
# standard input and produces runes.go on standard
# output.
# This script contains the tables necessary
# for rendering line art based on a byte map
# describing the thickness of each line.
#
# Line art has four possible edges and two possible
# weights: light and heavy.
# We denote these edges as north, south, east,
# and west, and capture two bits of information
# for each direction to indicate whether the
# corresponding line should be absent, light, or
# heavy.
# Since we have a domain of four possible input
# values and three possible representations,
# we use light lines to represent the middle 50% of
# the range, so heavy and absent represent outliers.
#
# For a one byte representation with the bits
# "NnSsEeWw", we construct a number in [0, 4) where:
# - 00 indicates absence.
# - 01 and 10 both indicate light lines.
# - 11 indicates a heavy line.
# Consequently, there are 256 possible
# inputs and 80 possible runes.
#
# runes.go captures two tables: one for the 80 runes,
# and another for the 256 possible input states and
# the indexes of the corresponding runes in the rune
# table.
#
# Each entry in runes.txt describes each of the 80
# runes and is of the form "N-S--e-w ╂" for every line
# art unicode rune annotated by whether each edge
# (north, south, east, west) is present, light, or
# heavy.
# Dash indicates absence.
# Miniscules indicate light.
# Majuscules indicate heavy.
#
# Like all Perl scripts, it is an expedient but
# heinous abomination.

my %table;
my %glyphs;
my %hints;
my $num = 0;
for (<>) {
    chomp;
    my $offset = $num++;
    my ($bits, $glyph) = split ' ';
    $glyphs[$offset] = $glyph || ' ';
    $hints[$offset] = $bits;
    my ($N, $n, $S, $s, $E, $e, $W, $w) = map {
        $_ ne '-' || 0;
    } split '', $bits;
    my @dims = map {
        my ($big, $small) = @$_;
        [$big ? ("11") : $small ? ("10", "01") : ("00")];
    } ([$N, $n], [$S, $s], [$E, $e], [$W, $w]);
    my @combos = ('');
    for my $dim (@dims) {
        @combos = map {
            my $head = $_;
            map {
                $head . $_;
            } @$dim;
        } @combos;
    }
    for my $address (map {
        unpack "C", pack "B[8]", $_;
    } @combos) {
        $table[$address] = $offset;
    }
}

printf "// Package rrr provides runes for rivers and ridges.\n";
printf "package rrr\n";
printf "\n";

printf "var lineArtRuneOffsets = []uint8{\n";
printf "\t// Index NnSsEeWs NnSsEeWw Rune\n";
for my $address (0..255) {
    my $offset = $table[$address];
    my $glyph = $glyphs[$offset];
    my $hint = $hints[$offset];
    printf "\t0x%02x, // %08b %s \"%s\"\n", $offset, $address, $hint, $glyph;
}
printf "}\n";

printf "\n";

printf "var lineArtRunes = []rune{\n";
for my $offset (0..80) {
    printf "\t'%s', // 0x%02x\n", $glyphs[$offset], $offset;
}
printf "}\n";
