package syzygy

// WDL (Win-Draw-Loss) results from Syzygy tablebases.
// These are used across the prober and the search integration.
const (
	WDLWin      = 2
	WDLBlessed  = 1
	WDLDraw     = 0
	WDLCursed   = -1
	WDLLoss     = -2
	WDLNotFound = -3
)
