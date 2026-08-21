package crawler

func clampStrategy(s Strategy) Strategy {
	if s.Workers <= 0 {
		s.Workers = 8
	}
	if s.Workers > 64 {
		s.Workers = 64
	}
	if s.MaxDepth <= 0 {
		s.MaxDepth = 4
	}
	if s.MaxDepth > 20 {
		s.MaxDepth = 20
	}
	if s.MaxPages <= 0 {
		s.MaxPages = 200
	}
	if s.MaxPages > 5000 {
		s.MaxPages = 5000
	}
	if s.UserAgent == "" {
		s.UserAgent = "MiniCrawl/1.0"
	}
	if s.GlobalRPS < 0 {
		s.GlobalRPS = 0
	}
	if s.HostRPS < 0 {
		s.HostRPS = 0
	}
	return s
}
