#!/usr/bin/python

#
# Generate test cases for version_test.go
#
# Herein lies my first ever python script...
#
import rpm

versions = [
  "",
  "0",
  "1",
  "2",
  "10",
  "100",
  "0.0",
  "0.1",
  "0.10",
  "0.99",
  "1.0",
  "1.99",
  "2.0",
  "0.0.0",
  "0.0.1",
  "0.0.2",
  "0.0.10",
  "0.0.99",
  "0.1.0",
  "0.2.0",
  "0.10.0",
  "0.99.0",
  "0.100.0",
  "0.0.0.0",
  "0.0.0.1",
  "0.0.0.10",
  "0.0.1.0",
  "0.0.01.0",
  "1.2.3.4",
  "1-2-3-4",
  "20150101",
  "20151212",
  "20151212.0",
  "20151212.1",
  "2015.1.1",
  "2015.02.02",
  "2015.12.12",
  "1.2.3a",
  "1.2.3b",
  "R16B",
  "R16C",
  "1.2.3.2016.1.1",
  "0.5a1.dev",
  "1.8.B59BrZX",
  "0.07b4p1",
  "3.99.5final.SP07",
  "3.99.5final.SP08",
  "0.4.tbb.20100203",
  "0.5.20120830CVS.el7",
  "1.el7",
  "1.el6",
  "10.el7",
  "01.el7",
  "0.17.20140318svn632.el7",
  "0.17.20140318svn633.el7",
  "1.20140522gitad6fb3e.el7",
  "1.20140522hitad6fb3e.el7",
  "8.20140605hgacf1c26e3019.el7",
  "8.20140605hgacf1c26e3029.el7",
  "22.svn457.el7",
  "22.svn458.el7",
]

print "\t// tests generated with version_test.py"
print "\ttests := []VerTest{"

for x in versions:
  for y in versions:
    print "\t\tVerTest{\"" + x + "\", \"" + y + "\",", rpm.labelCompare(("0", "0", x), ("0", "0", y)), "},"

print "\t}"
