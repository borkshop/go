COUNTRIES=$(
    jq -R 'select(startswith("#")|not)' countries.txt | jq -sc '
        [
            .[] |
            split("\t") as [$code, $iso3, $isonum, $isonam, $name, $captial, $area, $pop] |
            {$code, $name, pop: ($pop|tonumber)}
        ] as $countries |
        $countries | map(.pop) | add as $worldpop |
        $countries | map({
            key: .code,
            value: {
                name: .name,
                count: ((4000*.pop/$worldpop)|floor),
            }
        }) |
        from_entries
    '
)
jq -R . cities15000.txt | jq -scr \
    --argjson countries "$COUNTRIES" \
    --argjson admin1codes "$(
        jq -R . admin1codes.txt | jq -sc '
            [
                .[] |
                split("\t") as [$key, $value] |
                {$key, $value}
            ] |
            from_entries
        '
    )" \
'
    [
        .[] |
        split("\t") as [
            $geonameid,
            $name,
            $asciiname,
            $altnames,
            $lat,
            $lon,
            $featureclass,
            $featurecode,
            $countrycode,
            $cc2,
            $admin1code,
            $admin3code,
            $admin4code,
            $pop,
            $elev,
            $dem,
            $timezone,
            $modificationdate
        ] |
        {
            $name,
            $countrycode,
            country: $countries[$countrycode].name,
            region: $admin1codes[$countrycode + "." + $admin1code],
            pop: (try ($pop|tonumber) // 0),
        }
    ] |
    sort_by(-.pop) |
    group_by(.country)[] |
    .[0:$countries[.[0].countrycode].count][] |
    . as {$name, $country, $region} |
    [$name, $region, $country] |
    @csv
'
