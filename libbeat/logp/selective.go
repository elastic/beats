package logp

import (
	"go.uber.org/zap/zapcore"
)

type selectiveCore struct {
	allSelectors bool
	selectors    map[string]struct{}
	core         zapcore.Core
}

func selectiveWrapper(core zapcore.Core, selectors map[string]struct{}) zapcore.Core {
	if len(selectors) == 0 {
		return core
	}
	_, allSelectors := selectors["*"]
	return &selectiveCore{selectors: selectors, core: core, allSelectors: allSelectors}
}

// Enabled returns whether a given logging level is enabled when logging a
// message.
func (c *selectiveCore) Enabled(level zapcore.Level) bool {
	return c.core.Enabled(level)
}

// With adds structured context to the Core.
func (c *selectiveCore) With(fields []zapcore.Field) zapcore.Core {
	return selectiveWrapper(c.core.With(fields), c.selectors)
}

// Check determines whether the supplied Entry should be logged (using the
// embedded LevelEnabler and possibly some extra logic). If the entry
// should be logged, the Core adds itself to the CheckedEntry and returns
// the result.
//
// Callers must use Check before calling Write.
func (c *selectiveCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		if ent.Level == zapcore.DebugLevel {
			if c.allSelectors {
				return ce.AddCore(ent, c)
			} else if _, enabled := c.selectors[ent.LoggerName]; enabled {
				return ce.AddCore(ent, c)
			}
			return ce
		}

		return ce.AddCore(ent, c)
	}
	return ce
}

// Write serializes the Entry and any Fields supplied at the log site and
// writes them to their destination.
//
// If called, Write should always log the Entry and Fields; it should not
// replicate the logic of Check.
func (c *selectiveCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return c.core.Write(ent, fields)
}

// Sync flushes buffered logs (if any).
func (c *selectiveCore) Sync() error {
	return c.core.Sync()
}
