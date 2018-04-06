
//line parser.rl:1
package main


//line parser.go:7
const syslog_start int = 1
const syslog_first_final int = 69
const syslog_error int = 0

const syslog_en_main int = 1


//line parser.rl:8


// syslog
//<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8
//<13>Feb  5 17:32:18 10.0.0.99 Use the BFG!
func Parse(data []byte, syslog *SyslogMessage) {
    var p, cs int
    pe := len(data)
    
//line parser.go:25
	{
	cs = syslog_start
	}

//line parser.rl:17

    tok := 0
    eof := len(data)
    
//line parser.go:35
	{
	if ( p) == ( pe) {
		goto _test_eof
	}
	switch cs {
	case 1:
		goto st_case_1
	case 0:
		goto st_case_0
	case 2:
		goto st_case_2
	case 3:
		goto st_case_3
	case 4:
		goto st_case_4
	case 5:
		goto st_case_5
	case 6:
		goto st_case_6
	case 7:
		goto st_case_7
	case 8:
		goto st_case_8
	case 9:
		goto st_case_9
	case 10:
		goto st_case_10
	case 11:
		goto st_case_11
	case 12:
		goto st_case_12
	case 13:
		goto st_case_13
	case 14:
		goto st_case_14
	case 15:
		goto st_case_15
	case 16:
		goto st_case_16
	case 17:
		goto st_case_17
	case 18:
		goto st_case_18
	case 19:
		goto st_case_19
	case 20:
		goto st_case_20
	case 21:
		goto st_case_21
	case 22:
		goto st_case_22
	case 23:
		goto st_case_23
	case 24:
		goto st_case_24
	case 25:
		goto st_case_25
	case 26:
		goto st_case_26
	case 69:
		goto st_case_69
	case 70:
		goto st_case_70
	case 71:
		goto st_case_71
	case 72:
		goto st_case_72
	case 73:
		goto st_case_73
	case 74:
		goto st_case_74
	case 75:
		goto st_case_75
	case 76:
		goto st_case_76
	case 27:
		goto st_case_27
	case 28:
		goto st_case_28
	case 29:
		goto st_case_29
	case 30:
		goto st_case_30
	case 31:
		goto st_case_31
	case 32:
		goto st_case_32
	case 33:
		goto st_case_33
	case 34:
		goto st_case_34
	case 35:
		goto st_case_35
	case 36:
		goto st_case_36
	case 37:
		goto st_case_37
	case 38:
		goto st_case_38
	case 39:
		goto st_case_39
	case 40:
		goto st_case_40
	case 41:
		goto st_case_41
	case 42:
		goto st_case_42
	case 43:
		goto st_case_43
	case 44:
		goto st_case_44
	case 45:
		goto st_case_45
	case 46:
		goto st_case_46
	case 47:
		goto st_case_47
	case 48:
		goto st_case_48
	case 49:
		goto st_case_49
	case 50:
		goto st_case_50
	case 51:
		goto st_case_51
	case 52:
		goto st_case_52
	case 53:
		goto st_case_53
	case 54:
		goto st_case_54
	case 55:
		goto st_case_55
	case 56:
		goto st_case_56
	case 57:
		goto st_case_57
	case 58:
		goto st_case_58
	case 59:
		goto st_case_59
	case 60:
		goto st_case_60
	case 61:
		goto st_case_61
	case 62:
		goto st_case_62
	case 63:
		goto st_case_63
	case 64:
		goto st_case_64
	case 65:
		goto st_case_65
	case 66:
		goto st_case_66
	case 67:
		goto st_case_67
	case 68:
		goto st_case_68
	}
	goto st_out
	st_case_1:
		if data[( p)] == 60 {
			goto st2
		}
		goto st0
st_case_0:
	st0:
		cs = 0
		goto _out
	st2:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof2
		}
	st_case_2:
		if 48 <= data[( p)] && data[( p)] <= 57 {
			goto tr2
		}
		goto st0
tr2:
//line parser.rl:21

        tok = p
      
	goto st3
	st3:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof3
		}
	st_case_3:
//line parser.go:226
		if data[( p)] == 62 {
			goto tr4
		}
		if 48 <= data[( p)] && data[( p)] <= 57 {
			goto st4
		}
		goto st0
	st4:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof4
		}
	st_case_4:
		if data[( p)] == 62 {
			goto tr4
		}
		if 48 <= data[( p)] && data[( p)] <= 57 {
			goto st5
		}
		goto st0
	st5:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof5
		}
	st_case_5:
		if data[( p)] == 62 {
			goto tr4
		}
		if 48 <= data[( p)] && data[( p)] <= 57 {
			goto st6
		}
		goto st0
	st6:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof6
		}
	st_case_6:
		if data[( p)] == 62 {
			goto tr4
		}
		if 48 <= data[( p)] && data[( p)] <= 57 {
			goto st7
		}
		goto st0
	st7:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof7
		}
	st_case_7:
		if data[( p)] == 62 {
			goto tr4
		}
		goto st0
tr4:
//line parser.rl:25

        syslog.Priority = data[tok:p]
      
	goto st8
	st8:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof8
		}
	st_case_8:
//line parser.go:290
		switch data[( p)] {
		case 65:
			goto tr8
		case 68:
			goto tr9
		case 70:
			goto tr10
		case 74:
			goto tr11
		case 77:
			goto tr12
		case 78:
			goto tr13
		case 79:
			goto tr14
		case 83:
			goto tr15
		case 97:
			goto tr8
		case 100:
			goto tr9
		case 102:
			goto tr10
		case 106:
			goto tr11
		case 109:
			goto tr12
		case 110:
			goto tr13
		case 111:
			goto tr14
		case 115:
			goto tr15
		}
		goto st0
tr8:
//line parser.rl:21

        tok = p
      
	goto st9
	st9:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof9
		}
	st_case_9:
//line parser.go:337
		switch data[( p)] {
		case 112:
			goto st10
		case 117:
			goto st32
		}
		goto st0
	st10:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof10
		}
	st_case_10:
		if data[( p)] == 114 {
			goto st11
		}
		goto st0
	st11:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof11
		}
	st_case_11:
		switch data[( p)] {
		case 32:
			goto tr19
		case 105:
			goto st30
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
tr19:
//line parser.rl:33

        syslog.Month(data[tok:p])
      
	goto st12
	st12:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof12
		}
	st_case_12:
//line parser.go:380
		switch data[( p)] {
		case 32:
			goto st13
		case 51:
			goto tr23
		}
		switch {
		case data[( p)] < 49:
			if 9 <= data[( p)] && data[( p)] <= 13 {
				goto st13
			}
		case data[( p)] > 50:
			if 52 <= data[( p)] && data[( p)] <= 57 {
				goto tr24
			}
		default:
			goto tr22
		}
		goto st0
	st13:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof13
		}
	st_case_13:
		if 49 <= data[( p)] && data[( p)] <= 57 {
			goto tr24
		}
		goto st0
tr24:
//line parser.rl:21

        tok = p
      
	goto st14
	st14:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof14
		}
	st_case_14:
//line parser.go:420
		if data[( p)] == 32 {
			goto tr25
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr25
		}
		goto st0
tr25:
//line parser.rl:37

        syslog.Day(data[tok:p])
      
	goto st15
	st15:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof15
		}
	st_case_15:
//line parser.go:439
		if data[( p)] == 50 {
			goto tr27
		}
		if 48 <= data[( p)] && data[( p)] <= 49 {
			goto tr26
		}
		goto st0
tr26:
//line parser.rl:21

        tok = p
      
	goto st16
	st16:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof16
		}
	st_case_16:
//line parser.go:458
		if 48 <= data[( p)] && data[( p)] <= 57 {
			goto st17
		}
		goto st0
	st17:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof17
		}
	st_case_17:
		if data[( p)] == 58 {
			goto tr29
		}
		goto st0
tr29:
//line parser.rl:41

        syslog.Hour(data[tok:p])
      
	goto st18
	st18:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof18
		}
	st_case_18:
//line parser.go:483
		if 48 <= data[( p)] && data[( p)] <= 53 {
			goto tr30
		}
		goto st0
tr30:
//line parser.rl:21

        tok = p
      
	goto st19
	st19:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof19
		}
	st_case_19:
//line parser.go:499
		if 48 <= data[( p)] && data[( p)] <= 57 {
			goto st20
		}
		goto st0
	st20:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof20
		}
	st_case_20:
		if data[( p)] == 58 {
			goto tr32
		}
		goto st0
tr32:
//line parser.rl:45

        syslog.Minute(data[tok:p])
      
	goto st21
	st21:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof21
		}
	st_case_21:
//line parser.go:524
		if 48 <= data[( p)] && data[( p)] <= 53 {
			goto tr33
		}
		goto st0
tr33:
//line parser.rl:21

        tok = p
      
	goto st22
	st22:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof22
		}
	st_case_22:
//line parser.go:540
		if 48 <= data[( p)] && data[( p)] <= 57 {
			goto st23
		}
		goto st0
	st23:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof23
		}
	st_case_23:
		if data[( p)] == 32 {
			goto tr35
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr35
		}
		goto st0
tr35:
//line parser.rl:49

        syslog.Second(data[tok:p])
      
	goto st24
	st24:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof24
		}
	st_case_24:
//line parser.go:568
		switch {
		case data[( p)] > 95:
			if 97 <= data[( p)] && data[( p)] <= 122 {
				goto tr36
			}
		case data[( p)] >= 46:
			goto tr36
		}
		goto st0
tr36:
//line parser.rl:21

        tok = p
      
	goto st25
	st25:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof25
		}
	st_case_25:
//line parser.go:589
		if data[( p)] == 32 {
			goto tr37
		}
		switch {
		case data[( p)] < 46:
			if 9 <= data[( p)] && data[( p)] <= 13 {
				goto tr37
			}
		case data[( p)] > 95:
			if 97 <= data[( p)] && data[( p)] <= 122 {
				goto st25
			}
		default:
			goto st25
		}
		goto st0
tr37:
//line parser.rl:53

        syslog.Hostname = data[tok:p]
      
	goto st26
	st26:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof26
		}
	st_case_26:
//line parser.go:617
		switch data[( p)] {
		case 32:
			goto tr40
		case 91:
			goto tr40
		case 93:
			goto tr40
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr40
		}
		goto tr39
tr39:
//line parser.rl:21

        tok = p
      
	goto st69
	st69:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof69
		}
	st_case_69:
//line parser.go:641
		switch data[( p)] {
		case 32:
			goto st70
		case 58:
			goto tr74
		case 91:
			goto tr75
		case 93:
			goto st70
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto st70
		}
		goto st69
tr40:
//line parser.rl:21

        tok = p
      
	goto st70
	st70:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof70
		}
	st_case_70:
//line parser.go:667
		goto st70
tr74:
//line parser.rl:57

        syslog.Program = data[tok:p]
      
	goto st71
	st71:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof71
		}
	st_case_71:
//line parser.go:680
		switch data[( p)] {
		case 32:
			goto st72
		case 58:
			goto tr74
		case 91:
			goto tr75
		case 93:
			goto st70
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto st70
		}
		goto st69
	st72:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof72
		}
	st_case_72:
		goto tr40
tr75:
//line parser.rl:57

        syslog.Program = data[tok:p]
      
	goto st73
	st73:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof73
		}
	st_case_73:
//line parser.go:712
		if 48 <= data[( p)] && data[( p)] <= 57 {
			goto tr77
		}
		goto st70
tr77:
//line parser.rl:21

        tok = p
      
	goto st74
	st74:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof74
		}
	st_case_74:
//line parser.go:728
		if data[( p)] == 93 {
			goto tr79
		}
		if 48 <= data[( p)] && data[( p)] <= 57 {
			goto st74
		}
		goto st70
tr79:
//line parser.rl:61

        syslog.Pid = data[tok:p]
      
	goto st75
	st75:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof75
		}
	st_case_75:
//line parser.go:747
		if data[( p)] == 58 {
			goto st76
		}
		goto st70
	st76:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof76
		}
	st_case_76:
		if data[( p)] == 32 {
			goto st72
		}
		goto st70
tr27:
//line parser.rl:21

        tok = p
      
	goto st27
	st27:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof27
		}
	st_case_27:
//line parser.go:772
		if 48 <= data[( p)] && data[( p)] <= 51 {
			goto st17
		}
		goto st0
tr22:
//line parser.rl:21

        tok = p
      
	goto st28
	st28:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof28
		}
	st_case_28:
//line parser.go:788
		if data[( p)] == 32 {
			goto tr25
		}
		switch {
		case data[( p)] > 13:
			if 48 <= data[( p)] && data[( p)] <= 57 {
				goto st14
			}
		case data[( p)] >= 9:
			goto tr25
		}
		goto st0
tr23:
//line parser.rl:21

        tok = p
      
	goto st29
	st29:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof29
		}
	st_case_29:
//line parser.go:812
		if data[( p)] == 32 {
			goto tr25
		}
		switch {
		case data[( p)] > 13:
			if 48 <= data[( p)] && data[( p)] <= 49 {
				goto st14
			}
		case data[( p)] >= 9:
			goto tr25
		}
		goto st0
	st30:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof30
		}
	st_case_30:
		if data[( p)] == 108 {
			goto st31
		}
		goto st0
	st31:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof31
		}
	st_case_31:
		if data[( p)] == 32 {
			goto tr19
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
	st32:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof32
		}
	st_case_32:
		if data[( p)] == 103 {
			goto st33
		}
		goto st0
	st33:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof33
		}
	st_case_33:
		switch data[( p)] {
		case 32:
			goto tr19
		case 117:
			goto st34
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
	st34:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof34
		}
	st_case_34:
		if data[( p)] == 115 {
			goto st35
		}
		goto st0
	st35:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof35
		}
	st_case_35:
		if data[( p)] == 116 {
			goto st31
		}
		goto st0
tr9:
//line parser.rl:21

        tok = p
      
	goto st36
	st36:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof36
		}
	st_case_36:
//line parser.go:899
		if data[( p)] == 101 {
			goto st37
		}
		goto st0
	st37:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof37
		}
	st_case_37:
		if data[( p)] == 99 {
			goto st38
		}
		goto st0
	st38:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof38
		}
	st_case_38:
		switch data[( p)] {
		case 32:
			goto tr19
		case 101:
			goto st39
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
	st39:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof39
		}
	st_case_39:
		if data[( p)] == 109 {
			goto st40
		}
		goto st0
	st40:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof40
		}
	st_case_40:
		if data[( p)] == 98 {
			goto st41
		}
		goto st0
	st41:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof41
		}
	st_case_41:
		if data[( p)] == 101 {
			goto st42
		}
		goto st0
	st42:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof42
		}
	st_case_42:
		if data[( p)] == 114 {
			goto st31
		}
		goto st0
tr10:
//line parser.rl:21

        tok = p
      
	goto st43
	st43:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof43
		}
	st_case_43:
//line parser.go:975
		if data[( p)] == 101 {
			goto st44
		}
		goto st0
	st44:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof44
		}
	st_case_44:
		if data[( p)] == 98 {
			goto st45
		}
		goto st0
	st45:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof45
		}
	st_case_45:
		switch data[( p)] {
		case 32:
			goto tr19
		case 114:
			goto st46
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
	st46:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof46
		}
	st_case_46:
		if data[( p)] == 117 {
			goto st47
		}
		goto st0
	st47:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof47
		}
	st_case_47:
		if data[( p)] == 97 {
			goto st48
		}
		goto st0
	st48:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof48
		}
	st_case_48:
		if data[( p)] == 114 {
			goto st49
		}
		goto st0
	st49:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof49
		}
	st_case_49:
		if data[( p)] == 121 {
			goto st31
		}
		goto st0
tr11:
//line parser.rl:21

        tok = p
      
	goto st50
	st50:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof50
		}
	st_case_50:
//line parser.go:1051
		switch data[( p)] {
		case 97:
			goto st51
		case 117:
			goto st53
		}
		goto st0
	st51:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof51
		}
	st_case_51:
		if data[( p)] == 110 {
			goto st52
		}
		goto st0
	st52:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof52
		}
	st_case_52:
		switch data[( p)] {
		case 32:
			goto tr19
		case 117:
			goto st47
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
	st53:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof53
		}
	st_case_53:
		switch data[( p)] {
		case 108:
			goto st54
		case 110:
			goto st55
		}
		goto st0
	st54:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof54
		}
	st_case_54:
		switch data[( p)] {
		case 32:
			goto tr19
		case 121:
			goto st31
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
	st55:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof55
		}
	st_case_55:
		switch data[( p)] {
		case 32:
			goto tr19
		case 101:
			goto st31
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
tr12:
//line parser.rl:21

        tok = p
      
	goto st56
	st56:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof56
		}
	st_case_56:
//line parser.go:1136
		if data[( p)] == 97 {
			goto st57
		}
		goto st0
	st57:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof57
		}
	st_case_57:
		switch data[( p)] {
		case 32:
			goto tr19
		case 114:
			goto st58
		case 121:
			goto st31
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
	st58:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof58
		}
	st_case_58:
		switch data[( p)] {
		case 32:
			goto tr19
		case 99:
			goto st59
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
	st59:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof59
		}
	st_case_59:
		if data[( p)] == 104 {
			goto st31
		}
		goto st0
tr13:
//line parser.rl:21

        tok = p
      
	goto st60
	st60:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof60
		}
	st_case_60:
//line parser.go:1193
		if data[( p)] == 111 {
			goto st61
		}
		goto st0
	st61:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof61
		}
	st_case_61:
		if data[( p)] == 118 {
			goto st38
		}
		goto st0
tr14:
//line parser.rl:21

        tok = p
      
	goto st62
	st62:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof62
		}
	st_case_62:
//line parser.go:1218
		if data[( p)] == 99 {
			goto st63
		}
		goto st0
	st63:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof63
		}
	st_case_63:
		if data[( p)] == 116 {
			goto st64
		}
		goto st0
	st64:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof64
		}
	st_case_64:
		switch data[( p)] {
		case 32:
			goto tr19
		case 111:
			goto st40
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
tr15:
//line parser.rl:21

        tok = p
      
	goto st65
	st65:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof65
		}
	st_case_65:
//line parser.go:1258
		if data[( p)] == 101 {
			goto st66
		}
		goto st0
	st66:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof66
		}
	st_case_66:
		if data[( p)] == 112 {
			goto st67
		}
		goto st0
	st67:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof67
		}
	st_case_67:
		switch data[( p)] {
		case 32:
			goto tr19
		case 116:
			goto st68
		}
		if 9 <= data[( p)] && data[( p)] <= 13 {
			goto tr19
		}
		goto st0
	st68:
		if ( p)++; ( p) == ( pe) {
			goto _test_eof68
		}
	st_case_68:
		if data[( p)] == 101 {
			goto st39
		}
		goto st0
	st_out:
	_test_eof2: cs = 2; goto _test_eof
	_test_eof3: cs = 3; goto _test_eof
	_test_eof4: cs = 4; goto _test_eof
	_test_eof5: cs = 5; goto _test_eof
	_test_eof6: cs = 6; goto _test_eof
	_test_eof7: cs = 7; goto _test_eof
	_test_eof8: cs = 8; goto _test_eof
	_test_eof9: cs = 9; goto _test_eof
	_test_eof10: cs = 10; goto _test_eof
	_test_eof11: cs = 11; goto _test_eof
	_test_eof12: cs = 12; goto _test_eof
	_test_eof13: cs = 13; goto _test_eof
	_test_eof14: cs = 14; goto _test_eof
	_test_eof15: cs = 15; goto _test_eof
	_test_eof16: cs = 16; goto _test_eof
	_test_eof17: cs = 17; goto _test_eof
	_test_eof18: cs = 18; goto _test_eof
	_test_eof19: cs = 19; goto _test_eof
	_test_eof20: cs = 20; goto _test_eof
	_test_eof21: cs = 21; goto _test_eof
	_test_eof22: cs = 22; goto _test_eof
	_test_eof23: cs = 23; goto _test_eof
	_test_eof24: cs = 24; goto _test_eof
	_test_eof25: cs = 25; goto _test_eof
	_test_eof26: cs = 26; goto _test_eof
	_test_eof69: cs = 69; goto _test_eof
	_test_eof70: cs = 70; goto _test_eof
	_test_eof71: cs = 71; goto _test_eof
	_test_eof72: cs = 72; goto _test_eof
	_test_eof73: cs = 73; goto _test_eof
	_test_eof74: cs = 74; goto _test_eof
	_test_eof75: cs = 75; goto _test_eof
	_test_eof76: cs = 76; goto _test_eof
	_test_eof27: cs = 27; goto _test_eof
	_test_eof28: cs = 28; goto _test_eof
	_test_eof29: cs = 29; goto _test_eof
	_test_eof30: cs = 30; goto _test_eof
	_test_eof31: cs = 31; goto _test_eof
	_test_eof32: cs = 32; goto _test_eof
	_test_eof33: cs = 33; goto _test_eof
	_test_eof34: cs = 34; goto _test_eof
	_test_eof35: cs = 35; goto _test_eof
	_test_eof36: cs = 36; goto _test_eof
	_test_eof37: cs = 37; goto _test_eof
	_test_eof38: cs = 38; goto _test_eof
	_test_eof39: cs = 39; goto _test_eof
	_test_eof40: cs = 40; goto _test_eof
	_test_eof41: cs = 41; goto _test_eof
	_test_eof42: cs = 42; goto _test_eof
	_test_eof43: cs = 43; goto _test_eof
	_test_eof44: cs = 44; goto _test_eof
	_test_eof45: cs = 45; goto _test_eof
	_test_eof46: cs = 46; goto _test_eof
	_test_eof47: cs = 47; goto _test_eof
	_test_eof48: cs = 48; goto _test_eof
	_test_eof49: cs = 49; goto _test_eof
	_test_eof50: cs = 50; goto _test_eof
	_test_eof51: cs = 51; goto _test_eof
	_test_eof52: cs = 52; goto _test_eof
	_test_eof53: cs = 53; goto _test_eof
	_test_eof54: cs = 54; goto _test_eof
	_test_eof55: cs = 55; goto _test_eof
	_test_eof56: cs = 56; goto _test_eof
	_test_eof57: cs = 57; goto _test_eof
	_test_eof58: cs = 58; goto _test_eof
	_test_eof59: cs = 59; goto _test_eof
	_test_eof60: cs = 60; goto _test_eof
	_test_eof61: cs = 61; goto _test_eof
	_test_eof62: cs = 62; goto _test_eof
	_test_eof63: cs = 63; goto _test_eof
	_test_eof64: cs = 64; goto _test_eof
	_test_eof65: cs = 65; goto _test_eof
	_test_eof66: cs = 66; goto _test_eof
	_test_eof67: cs = 67; goto _test_eof
	_test_eof68: cs = 68; goto _test_eof

	_test_eof: {}
	if ( p) == eof {
		switch cs {
		case 69, 70, 71, 72, 73, 74, 75, 76:
//line parser.rl:29

        syslog.Message = data[tok:p]
      
//line parser.go:1381
		}
	}

	_out: {}
	}

//line parser.rl:108

}
