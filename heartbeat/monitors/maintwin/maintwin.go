package maintwin

import (
	"fmt"
	"time"

	"github.com/gorhill/cronexpr"
)

type MaintWin struct {
	Zone     string        `config:"zone" validate:"required"`
	Start    string        `config:"start" validate:"required"`
	Duration time.Duration `config:"duration" validate:"required"`
}

func (mw *MaintWin) Parse() (pmw ParsedMaintWin, err error) {
	pmw = ParsedMaintWin{Duration: mw.Duration}

	pmw.Location, err = time.LoadLocation(mw.Zone)
	if err != nil {
		return ParsedMaintWin{}, fmt.Errorf("could not load zone '%s': %w", mw.Zone, err)
	}

	pmw.StartExpr, err = cronexpr.Parse(mw.Start)
	if err != nil {
		return ParsedMaintWin{}, fmt.Errorf("could not parse expr start '%s': %w", mw.Start, err)
	}
	nextStarts := pmw.StartExpr.NextN(time.Now(), 2)
	pmw.TimeBetweenStarts = nextStarts[1].Sub(nextStarts[0])
	pmw.NonMaintWinDuration = pmw.TimeBetweenStarts - pmw.Duration
	return pmw, nil
}

type ParsedMaintWin struct {
	Location            *time.Location
	StartExpr           *cronexpr.Expression
	Duration            time.Duration
	TimeBetweenStarts   time.Duration
	NonMaintWinDuration time.Duration
}

func (pmw ParsedMaintWin) IsActive(tOrig time.Time) bool {
	t := tOrig.In(pmw.Location)
	// this is confusing to understand, no alternative to staring at it a bit
	nextStart := pmw.StartExpr.Next(t)
	timeTilNextStart := t.Sub(nextStart.Add(-pmw.TimeBetweenStarts))

	doesMatch := timeTilNextStart <= pmw.Duration
	return doesMatch
}
