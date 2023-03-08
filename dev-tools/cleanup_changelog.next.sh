# Source this file from the root of the beats repository
cat CHANGELOG.asciidoc | perl -n -e'/\{pull\}(\d+)\[(\d+)\]/ && print "{pull}$1\[$2\]\n"' > changelog-exps.txt
cat CHANGELOG.asciidoc | perl -n -e'/\{issue\}(\d+)\[(\d+)\]/ && print "{issue}$1\[$2\]\n"' >> changelog-exps.txt
grep -f changelog-exps.txt -F -v CHANGELOG.next.asciidoc > CHANGELOG.next.clean.asciidoc
mv CHANGELOG.next.clean.asciidoc CHANGELOG.next.asciidoc
rm changelog-exps.txt
