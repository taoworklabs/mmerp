package accounting

import "github.com/taoworklabs/mmerp/internal/platform"

func Module(s *Service) platform.Module {
	return platform.Module{Name: "accounting", Product: "accounting"}
}
