package domain

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// Sector 是行业分类的业务概念。
type Sector struct {
	Code string
	Name string
}

// Validate 检查行业分类的最小领域约束。
func (sector Sector) Validate() error {
	if strings.TrimSpace(sector.Code) == "" {
		return errors.New("sector code is required")
	}
	if strings.TrimSpace(sector.Name) == "" {
		return errors.New("sector name is required")
	}
	return nil
}

// SectorMembership 描述股票属于哪个行业。
type SectorMembership struct {
	SectorCode     string
	InstrumentCode string
}

// Validate 检查行业成分关系的最小领域约束。
func (membership SectorMembership) Validate() error {
	if strings.TrimSpace(membership.SectorCode) == "" {
		return errors.New("sector membership sector code is required")
	}
	if strings.TrimSpace(membership.InstrumentCode) == "" {
		return errors.New("sector membership instrument code is required")
	}
	return nil
}

// SectorComponent 是行业聚合所需的一只成分股前后收盘价。
type SectorComponent struct {
	SectorCode     string
	SectorName     string
	InstrumentCode string
	InstrumentName string
	TradeDate      string
	PreviousClose  string
	CurrentClose   string
}

// Validate 检查行业成分行情的完整性。
func (component SectorComponent) Validate() error {
	for name, value := range map[string]string{
		"sector code": component.SectorCode, "sector name": component.SectorName,
		"instrument code": component.InstrumentCode, "instrument name": component.InstrumentName,
		"trade date": component.TradeDate,
	} {
		if strings.TrimSpace(value) == "" {
			return errors.New("sector component " + name + " is required")
		}
	}
	if _, err := positiveDecimal(component.PreviousClose, "previous close"); err != nil {
		return err
	}
	if _, err := positiveDecimal(component.CurrentClose, "current close"); err != nil {
		return err
	}
	return nil
}

// SectorLeader 是行业内涨幅最高的股票。
type SectorLeader struct {
	Code          string
	Name          string
	ChangePercent string
}

// SectorPerformance 是一个行业在某交易日的聚合表现。
type SectorPerformance struct {
	Code           string
	Name           string
	ChangePercent  string
	ComponentCount int64
	Leader         SectorLeader
}

type sectorPerformanceGroup struct {
	code         string
	name         string
	tradeDate    string
	components   map[string]struct{}
	returnTotal  *big.Rat
	leader       SectorLeader
	leaderReturn *big.Rat
}

// AggregateSectorPerformances 按成分股当日涨跌幅等权计算行业表现。
func AggregateSectorPerformances(components []SectorComponent) ([]SectorPerformance, error) {
	if len(components) == 0 {
		return nil, errors.New("sector components are required")
	}

	groups := make(map[string]*sectorPerformanceGroup)
	for _, component := range components {
		if err := component.Validate(); err != nil {
			return nil, err
		}
		changePercent, err := componentChangePercent(component)
		if err != nil {
			return nil, fmt.Errorf("calculate sector component %q: %w", component.InstrumentCode, err)
		}

		group, exists := groups[component.SectorCode]
		if !exists {
			group = &sectorPerformanceGroup{
				code:         component.SectorCode,
				name:         component.SectorName,
				tradeDate:    component.TradeDate,
				components:   make(map[string]struct{}),
				returnTotal:  new(big.Rat),
				leaderReturn: changePercent,
				leader:       SectorLeader{Code: component.InstrumentCode, Name: component.InstrumentName, ChangePercent: formatPercent(changePercent)},
			}
			groups[component.SectorCode] = group
		}
		if group.name != component.SectorName || group.tradeDate != component.TradeDate {
			return nil, fmt.Errorf("sector %q has inconsistent name or trade date", component.SectorCode)
		}
		if _, exists := group.components[component.InstrumentCode]; exists {
			return nil, fmt.Errorf("sector %q contains duplicate instrument %q", component.SectorCode, component.InstrumentCode)
		}
		group.components[component.InstrumentCode] = struct{}{}
		group.returnTotal.Add(group.returnTotal, changePercent)
		if changePercent.Cmp(group.leaderReturn) > 0 ||
			(changePercent.Cmp(group.leaderReturn) == 0 && component.InstrumentCode < group.leader.Code) {
			group.leaderReturn = changePercent
			group.leader = SectorLeader{Code: component.InstrumentCode, Name: component.InstrumentName, ChangePercent: formatPercent(changePercent)}
		}
	}

	sectorCodes := make([]string, 0, len(groups))
	for code := range groups {
		sectorCodes = append(sectorCodes, code)
	}
	sort.Strings(sectorCodes)

	performances := make([]SectorPerformance, 0, len(sectorCodes))
	for _, code := range sectorCodes {
		group := groups[code]
		average := new(big.Rat).Quo(group.returnTotal, new(big.Rat).SetInt64(int64(len(group.components))))
		performances = append(performances, SectorPerformance{
			Code:           group.code,
			Name:           group.name,
			ChangePercent:  formatPercent(average),
			ComponentCount: int64(len(group.components)),
			Leader:         group.leader,
		})
	}
	return performances, nil
}

func componentChangePercent(component SectorComponent) (*big.Rat, error) {
	previous, err := positiveDecimal(component.PreviousClose, "previous close")
	if err != nil {
		return nil, err
	}
	current, err := positiveDecimal(component.CurrentClose, "current close")
	if err != nil {
		return nil, err
	}
	change := new(big.Rat).Sub(current, previous)
	change.Quo(change, previous)
	return change.Mul(change, big.NewRat(100, 1)), nil
}

func positiveDecimal(value, label string) (*big.Rat, error) {
	parsed, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok {
		return nil, fmt.Errorf("sector component %s must be a decimal", label)
	}
	if parsed.Sign() <= 0 {
		return nil, fmt.Errorf("sector component %s must be positive", label)
	}
	return parsed, nil
}

func formatPercent(value *big.Rat) string {
	scaled := new(big.Rat).Mul(value, big.NewRat(100, 1))
	quotient, remainder := new(big.Int).QuoRem(scaled.Num(), scaled.Denom(), new(big.Int))
	if remainder.Sign() != 0 {
		twiceRemainder := new(big.Int).Lsh(new(big.Int).Abs(remainder), 1)
		if twiceRemainder.Cmp(scaled.Denom()) >= 0 {
			if scaled.Num().Sign() < 0 {
				quotient.Sub(quotient, big.NewInt(1))
			} else {
				quotient.Add(quotient, big.NewInt(1))
			}
		}
	}

	sign := ""
	if quotient.Sign() < 0 {
		sign = "-"
	}
	absQuotient := new(big.Int).Abs(quotient)
	major := new(big.Int).Quo(absQuotient, big.NewInt(100))
	minor := new(big.Int).Mod(absQuotient, big.NewInt(100))
	return fmt.Sprintf("%s%s.%02d", sign, major.String(), minor.Int64())
}
