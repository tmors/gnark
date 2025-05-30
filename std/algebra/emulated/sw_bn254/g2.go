package sw_bn254

import (
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/algebra/emulated/fields_bn254"
	"github.com/consensys/gnark/std/math/emulated"
)

type G2 struct {
	api frontend.API
	fp  *emulated.Field[BaseField]
	*fields_bn254.Ext2
	w    *emulated.Element[BaseField]
	u, v *fields_bn254.E2
}

type g2AffP struct {
	X, Y fields_bn254.E2
}

// G2Affine represents G2 element with optional embedded line precomputations.
type G2Affine struct {
	P     g2AffP
	Lines *lineEvaluations
}

func newG2AffP(v bn254.G2Affine) g2AffP {
	return g2AffP{
		X: fields_bn254.E2{
			A0: emulated.ValueOf[BaseField](v.X.A0),
			A1: emulated.ValueOf[BaseField](v.X.A1),
		},
		Y: fields_bn254.E2{
			A0: emulated.ValueOf[BaseField](v.Y.A0),
			A1: emulated.ValueOf[BaseField](v.Y.A1),
		},
	}
}

func NewG2(api frontend.API) (*G2, error) {
	fp, err := emulated.NewField[BaseField](api)
	if err != nil {
		return nil, fmt.Errorf("new base api: %w", err)
	}
	w := emulated.ValueOf[BaseField]("21888242871839275220042445260109153167277707414472061641714758635765020556616")
	u := fields_bn254.E2{
		A0: emulated.ValueOf[BaseField]("21575463638280843010398324269430826099269044274347216827212613867836435027261"),
		A1: emulated.ValueOf[BaseField]("10307601595873709700152284273816112264069230130616436755625194854815875713954"),
	}
	v := fields_bn254.E2{
		A0: emulated.ValueOf[BaseField]("2821565182194536844548159561693502659359617185244120367078079554186484126554"),
		A1: emulated.ValueOf[BaseField]("3505843767911556378687030309984248845540243509899259641013678093033130930403"),
	}
	return &G2{
		api:  api,
		fp:   fp,
		Ext2: fields_bn254.NewExt2(api),
		w:    &w,
		u:    &u,
		v:    &v,
	}, nil
}

func NewG2Affine(v bn254.G2Affine) G2Affine {
	return G2Affine{
		P: newG2AffP(v),
	}
}

// NewG2AffineFixed returns witness of v with precomputations for efficient
// pairing computation.
func NewG2AffineFixed(v bn254.G2Affine) G2Affine {
	if !v.IsInSubGroup() {
		// for the pairing check we check that G2 point is already in the
		// subgroup when we compute the lines in circuit. However, when the
		// point is given as a constant, then we already precompute the lines at
		// circuit compile time without explicitly checking the G2 membership.
		// So, we need to check that the point is in the subgroup before we
		// compute the lines.
		panic("given point is not in the G2 subgroup")
	}
	lines := precomputeLines(v)
	return G2Affine{
		P:     newG2AffP(v),
		Lines: &lines,
	}
}

// NewG2AffineFixedPlaceholder returns a placeholder for the circuit compilation
// when witness will be given with line precomputations using
// [NewG2AffineFixed].
func NewG2AffineFixedPlaceholder() G2Affine {
	var lines lineEvaluations
	for i := 0; i < len(bn254.LoopCounter); i++ {
		lines[0][i] = &lineEvaluation{}
		lines[1][i] = &lineEvaluation{}
	}
	return G2Affine{
		Lines: &lines,
	}
}

func (g2 *G2) phi(q *G2Affine) *G2Affine {
	x := g2.Ext2.MulByElement(&q.P.X, g2.w)

	return &G2Affine{
		P: g2AffP{
			X: *x,
			Y: *g2.Ext2.Neg(&q.P.Y),
		},
	}
}

func (g2 *G2) psi(q *G2Affine) *G2Affine {
	x := g2.Ext2.Conjugate(&q.P.X)
	x = g2.Ext2.Mul(x, g2.u)
	y := g2.Ext2.Conjugate(&q.P.Y)
	y = g2.Ext2.Mul(y, g2.v)

	return &G2Affine{
		P: g2AffP{
			X: *x,
			Y: *y,
		},
	}
}

func (g2 *G2) scalarMulBySeed(q *G2Affine) *G2Affine {
	z := g2.double(q)
	t0 := g2.add(q, z)
	t2 := g2.add(q, t0)
	t1 := g2.add(z, t2)
	z = g2.doubleAndAdd(t1, t0)
	t0 = g2.add(t0, z)
	t2 = g2.add(t2, t0)
	t1 = g2.add(t1, t2)
	t0 = g2.add(t0, t1)
	t1 = g2.add(t1, t0)
	t0 = g2.add(t0, t1)
	t2 = g2.add(t2, t0)
	t1 = g2.doubleAndAdd(t2, t1)
	t2 = g2.add(t2, t1)
	z = g2.add(z, t2)
	t2 = g2.add(t2, z)
	z = g2.doubleAndAdd(t2, z)
	t0 = g2.add(t0, z)
	t1 = g2.add(t1, t0)
	t3 := g2.double(t1)
	t3 = g2.doubleAndAdd(t3, t1)
	t2 = g2.add(t2, t3)
	t1 = g2.add(t1, t2)
	t2 = g2.add(t2, t1)
	t2 = g2.doubleN(t2, 16)
	t1 = g2.doubleAndAdd(t2, t1)
	t1 = g2.doubleN(t1, 13)
	t0 = g2.doubleAndAdd(t1, t0)
	t0 = g2.doubleN(t0, 15)
	z = g2.doubleAndAdd(t0, z)

	return z
}

func (g2 G2) add(p, q *G2Affine) *G2Affine {
	mone := g2.fp.NewElement(-1)

	// compute λ = (q.y-p.y)/(q.x-p.x)
	qypy := g2.Ext2.Sub(&q.P.Y, &p.P.Y)
	qxpx := g2.Ext2.Sub(&q.P.X, &p.P.X)
	λ := g2.Ext2.DivUnchecked(qypy, qxpx)

	// xr = λ²-p.x-q.x
	xr0 := g2.fp.Eval([][]*baseEl{{&λ.A0, &λ.A0}, {mone, &λ.A1, &λ.A1}, {mone, &p.P.X.A0}, {mone, &q.P.X.A0}}, []int{1, 1, 1, 1})
	xr1 := g2.fp.Eval([][]*baseEl{{&λ.A0, &λ.A1}, {mone, &p.P.X.A1}, {mone, &q.P.X.A1}}, []int{2, 1, 1})
	xr := &fields_bn254.E2{A0: *xr0, A1: *xr1}

	// p.y = λ(p.x-r.x) - p.y
	yr := g2.Ext2.Sub(&p.P.X, xr)
	yr0 := g2.fp.Eval([][]*baseEl{{&λ.A0, &yr.A0}, {mone, &λ.A1, &yr.A1}, {mone, &p.P.Y.A0}}, []int{1, 1, 1})
	yr1 := g2.fp.Eval([][]*baseEl{{&λ.A0, &yr.A1}, {&λ.A1, &yr.A0}, {mone, &p.P.Y.A1}}, []int{1, 1, 1})
	yr = &fields_bn254.E2{A0: *yr0, A1: *yr1}

	return &G2Affine{
		P: g2AffP{
			X: *xr,
			Y: *yr,
		},
	}
}

func (g2 G2) neg(p *G2Affine) *G2Affine {
	xr := &p.P.X
	yr := g2.Ext2.Neg(&p.P.Y)
	return &G2Affine{
		P: g2AffP{
			X: *xr,
			Y: *yr,
		},
	}
}

func (g2 G2) sub(p, q *G2Affine) *G2Affine {
	qNeg := g2.neg(q)
	return g2.add(p, qNeg)
}

func (g2 *G2) double(p *G2Affine) *G2Affine {
	mone := g2.fp.NewElement(-1)

	// compute λ = (3p.x²)/2*p.y
	xx3a := g2.Square(&p.P.X)
	xx3a = g2.MulByConstElement(xx3a, big.NewInt(3))
	y2 := g2.Double(&p.P.Y)
	λ := g2.DivUnchecked(xx3a, y2)

	// xr = λ²-2p.x
	xr0 := g2.fp.Eval([][]*baseEl{{&λ.A0, &λ.A0}, {mone, &λ.A1, &λ.A1}, {mone, &p.P.X.A0}}, []int{1, 1, 2})
	xr1 := g2.fp.Eval([][]*baseEl{{&λ.A0, &λ.A1}, {mone, &p.P.X.A1}}, []int{2, 2})
	xr := &fields_bn254.E2{A0: *xr0, A1: *xr1}

	// yr = λ(p-xr) - p.y
	yr := g2.Ext2.Sub(&p.P.X, xr)
	yr0 := g2.fp.Eval([][]*baseEl{{&λ.A0, &yr.A0}, {mone, &λ.A1, &yr.A1}, {mone, &p.P.Y.A0}}, []int{1, 1, 1})
	yr1 := g2.fp.Eval([][]*baseEl{{&λ.A0, &yr.A1}, {&λ.A1, &yr.A0}, {mone, &p.P.Y.A1}}, []int{1, 1, 1})
	yr = &fields_bn254.E2{A0: *yr0, A1: *yr1}

	return &G2Affine{
		P: g2AffP{
			X: *xr,
			Y: *yr,
		},
	}
}

func (g2 *G2) doubleN(p *G2Affine, n int) *G2Affine {
	pn := p
	for s := 0; s < n; s++ {
		pn = g2.double(pn)
	}
	return pn
}

func (g2 G2) doubleAndAdd(p, q *G2Affine) *G2Affine {
	mone := g2.fp.NewElement(-1)

	// compute λ1 = (q.y-p.y)/(q.x-p.x)
	yqyp := g2.Ext2.Sub(&q.P.Y, &p.P.Y)
	xqxp := g2.Ext2.Sub(&q.P.X, &p.P.X)
	λ1 := g2.Ext2.DivUnchecked(yqyp, xqxp)

	// compute x2 = λ1²-p.x-q.x
	x20 := g2.fp.Eval([][]*baseEl{{&λ1.A0, &λ1.A0}, {mone, &λ1.A1, &λ1.A1}, {mone, &p.P.X.A0}, {mone, &q.P.X.A0}}, []int{1, 1, 1, 1})
	x21 := g2.fp.Eval([][]*baseEl{{&λ1.A0, &λ1.A1}, {mone, &p.P.X.A1}, {mone, &q.P.X.A1}}, []int{2, 1, 1})
	x2 := &fields_bn254.E2{A0: *x20, A1: *x21}

	// omit y2 computation
	// compute -λ2 = λ1+2*p.y/(x2-p.x)
	ypyp := g2.Ext2.Add(&p.P.Y, &p.P.Y)
	x2xp := g2.Ext2.Sub(x2, &p.P.X)
	λ2 := g2.Ext2.DivUnchecked(ypyp, x2xp)
	λ2 = g2.Ext2.Add(λ1, λ2)

	// compute x3 = (-λ2)²-p.x-x2
	x30 := g2.fp.Eval([][]*baseEl{{&λ2.A0, &λ2.A0}, {mone, &λ2.A1, &λ2.A1}, {mone, &p.P.X.A0}, {mone, x20}}, []int{1, 1, 1, 1})
	x31 := g2.fp.Eval([][]*baseEl{{&λ2.A0, &λ2.A1}, {mone, &p.P.X.A1}, {mone, x21}}, []int{2, 1, 1})
	x3 := &fields_bn254.E2{A0: *x30, A1: *x31}

	// compute y3 = -λ2*(x3 - p.x)-p.y
	y3 := g2.Ext2.Sub(x3, &p.P.X)
	y30 := g2.fp.Eval([][]*baseEl{{&λ2.A0, &y3.A0}, {mone, &λ2.A1, &y3.A1}, {mone, &p.P.Y.A0}}, []int{1, 1, 1})
	y31 := g2.fp.Eval([][]*baseEl{{&λ2.A0, &y3.A1}, {&λ2.A1, &y3.A0}, {mone, &p.P.Y.A1}}, []int{1, 1, 1})
	y3 = &fields_bn254.E2{A0: *y30, A1: *y31}

	return &G2Affine{
		P: g2AffP{
			X: *x3,
			Y: *y3,
		},
	}
}

// AssertIsEqual asserts that p and q are the same point.
func (g2 *G2) AssertIsEqual(p, q *G2Affine) {
	g2.Ext2.AssertIsEqual(&p.P.X, &q.P.X)
	g2.Ext2.AssertIsEqual(&p.P.Y, &q.P.Y)
}

func (g2 *G2) IsEqual(p, q *G2Affine) frontend.Variable {
	xEqual := g2.Ext2.IsEqual(&p.P.X, &q.P.X)
	yEqual := g2.Ext2.IsEqual(&p.P.Y, &q.P.Y)
	return g2.api.And(xEqual, yEqual)
}
