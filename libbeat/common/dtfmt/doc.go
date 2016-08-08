// dtfmt package provides time formatter support with pattern syntax mostly
// similar to joda DateTimeFormat. The pattern syntax supported is a subset
// (mostly compatible) with joda DateTimeFormat.
//
//
//  Symbol  Meaning                      Type     Supported Examples
//  ------  -------                      -------  --------- -------
//  G       era                          text      no       AD
//  C       century of era (&gt;=0)      number    no       20
//  Y       year of era (&gt;=0)         year      yes      1996
//
//  x       weekyear                     year      yes      1996
//  w       week of weekyear             number    yes      27
//  e       day of week                  number    yes      2
//  E       day of week                  text      yes      Tuesday; Tue
//
//  y       year                         year      yes      1996
//  D       day of year                  number    yes      189
//  M       month of year                month     yes      July; Jul; 07
//  d       day of month                 number    yes      10
//
//  a       halfday of day               text      yes      PM
//  K       hour of halfday (0~11)       number    yes      0
//  h       clockhour of halfday (1~12)  number    yes      12
//
//  H       hour of day (0~23)           number    yes      0
//  k       clockhour of day (1~24)      number    yes      24
//  m       minute of hour               number    yes      30
//  s       second of minute             number    yes      55
//  S       fraction of second           millis    no       978
//
//  z       time zone                    text      no       Pacific Standard Time; PST
//  Z       time zone offset/id          zone      no       -0800; -08:00; America/Los_Angeles
//
//  '       escape for text              delimiter
//  ''      single quote                 literal
//
// The format is based on pattern letter count. Any character not in the range
// [a-z][A-Z] is interpreted as literal and copied into final string as is.
// Arbitrary Literals can also be written using single quotes `'`
//
//  Types:          Notes:
//  ------          ------
//   text           Use full form if number of letters is >= 4.
//                  Otherwise a short form is used (if available).
//
//   number         Minimum number of digits depends on number of letters.
//                  Shorter numbers are zero-padded.
//
//   year           mostly like number. If Pattern length is 2,
//                  the year will be displayed as zero-based year
//                  of the century (modulo 100)
//
//   month          If pattern length >= 3, formatting is according to
//                  text type. Otherwise number type
//                  formatting rules are applied.
//
//   millis         Not yet supported
//
//   zone           Not yet supported
//
//   literal        Literals are copied as is into formatted string
//
package dtfmt
