package compile

import (
	"matrixone/pkg/sql/colexec/mergetop"
	vtop "matrixone/pkg/sql/colexec/top"
	"matrixone/pkg/sql/colexec/transfer"
	"matrixone/pkg/sql/op/top"
	"matrixone/pkg/vm"
	"matrixone/pkg/vm/mmu/guest"
	"matrixone/pkg/vm/process"
	"sync"
)

func (c *compile) compileTop(o *top.Top, mp map[string]uint64) ([]*Scope, error) {
	{
		for _, g := range o.Gs {
			mp[g.Name]++
		}
	}
	ss, err := c.compile(o.Prev, mp)
	if err != nil {
		return nil, err
	}
	rs := new(Scope)
	gm := guest.New(c.proc.Gm.Limit, c.proc.Gm.Mmu)
	rs.Proc = process.New(gm, c.proc.Mp)
	rs.Proc.Lim = c.proc.Lim
	rs.Proc.Reg.Ws = make([]*process.WaitRegister, len(ss))
	{
		for i, j := 0, len(ss); i < j; i++ {
			rs.Proc.Reg.Ws[i] = &process.WaitRegister{
				Wg: new(sync.WaitGroup),
				Ch: make(chan interface{}),
			}
		}
	}
	fs := make([]vtop.Field, len(o.Gs))
	{
		for i, g := range o.Gs {
			fs[i] = vtop.Field{
				Attr: g.Name,
				Type: vtop.Direction(g.Dirt),
			}
		}
	}
	for i, s := range ss {
		s.Ins = append(s.Ins, vm.Instruction{
			Op: vm.Top,
			Arg: &vtop.Argument{
				Fs:    fs,
				Limit: o.Limit,
			},
		})
		s.Ins = append(s.Ins, vm.Instruction{
			Op: vm.Transfer,
			Arg: &transfer.Argument{
				Mmu: gm,
				Reg: rs.Proc.Reg.Ws[i],
			},
		})
	}
	rs.Ss = ss
	rs.Magic = Merge
	mfs := make([]mergetop.Field, len(o.Gs))
	{
		for i, g := range o.Gs {
			mfs[i] = mergetop.Field{
				Attr: g.Name,
				Type: mergetop.Direction(g.Dirt),
			}
		}
	}
	rs.Ins = append(rs.Ins, vm.Instruction{
		Op: vm.MergeTop,
		Arg: &mergetop.Argument{
			Fs:    mfs,
			Limit: o.Limit,
		},
	})
	return []*Scope{rs}, nil
}