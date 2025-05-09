package gkr

import (
	"fmt"
	"hash"
	"math/rand"
	"strconv"
	"testing"
	"time"

	gcHash "github.com/consensys/gnark-crypto/hash"

	bls12377 "github.com/consensys/gnark/constraint/bls12-377"
	bls12381 "github.com/consensys/gnark/constraint/bls12-381"
	bls24315 "github.com/consensys/gnark/constraint/bls24-315"
	bls24317 "github.com/consensys/gnark/constraint/bls24-317"
	bw6633 "github.com/consensys/gnark/constraint/bw6-633"
	bw6761 "github.com/consensys/gnark/constraint/bw6-761"
	"github.com/consensys/gnark/test"

	bn254 "github.com/consensys/gnark/constraint/bn254"
	"github.com/stretchr/testify/require"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	gkr "github.com/consensys/gnark/internal/gkr/bn254"
	stdHash "github.com/consensys/gnark/std/hash"
	"github.com/consensys/gnark/std/hash/mimc"
)

// compressThreshold --> if linear expressions are larger than this, the frontend will introduce
// intermediate constraints. The lower this number is, the faster compile time should be (to a point)
// but resulting circuit will have more constraints (slower proving time).
const compressThreshold = 1000

type doubleNoDependencyCircuit struct {
	X        []frontend.Variable
	hashName string
}

func (c *doubleNoDependencyCircuit) Define(api frontend.API) error {
	gkr := NewApi()
	var x constraint.GkrVariable
	var err error
	if x, err = gkr.Import(c.X); err != nil {
		return err
	}
	z := gkr.Add(x, x)
	var solution Solution
	if solution, err = gkr.Solve(api); err != nil {
		return err
	}
	Z := solution.Export(z)

	for i := range Z {
		api.AssertIsEqual(Z[i], api.Mul(2, c.X[i]))
	}

	return solution.Verify(c.hashName)
}

func TestDoubleNoDependencyCircuit(t *testing.T) {
	assert := test.NewAssert(t)

	xValuess := [][]frontend.Variable{
		{1, 1},
		{1, 2},
	}

	hashes := []string{"-1", "-20"}

	for i, xValues := range xValuess {
		for _, hashName := range hashes {
			assignment := doubleNoDependencyCircuit{X: xValues}
			circuit := doubleNoDependencyCircuit{X: make([]frontend.Variable, len(xValues)), hashName: hashName}
			assert.Run(func(assert *test.Assert) {
				assert.CheckCircuit(&circuit, test.WithValidAssignment(&assignment), test.WithCurves(ecc.BN254))
			}, fmt.Sprintf("xValue=%d/hash=%s", i, hashName))

		}
	}
}

type sqNoDependencyCircuit struct {
	X        []frontend.Variable
	hashName string
}

func (c *sqNoDependencyCircuit) Define(api frontend.API) error {
	gkr := NewApi()
	var x constraint.GkrVariable
	var err error
	if x, err = gkr.Import(c.X); err != nil {
		return err
	}
	z := gkr.Mul(x, x)
	var solution Solution
	if solution, err = gkr.Solve(api); err != nil {
		return err
	}
	Z := solution.Export(z)

	for i := range Z {
		api.AssertIsEqual(Z[i], api.Mul(c.X[i], c.X[i]))
	}

	return solution.Verify(c.hashName)
}

func TestSqNoDependencyCircuit(t *testing.T) {
	assert := test.NewAssert(t)

	xValuess := [][]frontend.Variable{
		{1, 1},
		{1, 2},
	}

	hashes := []string{"-1", "-20"}

	for i, xValues := range xValuess {
		for _, hashName := range hashes {
			assignment := sqNoDependencyCircuit{X: xValues}
			circuit := sqNoDependencyCircuit{X: make([]frontend.Variable, len(xValues)), hashName: hashName}
			assert.Run(func(assert *test.Assert) {
				assert.CheckCircuit(&circuit, test.WithValidAssignment(&assignment), test.WithCurves(ecc.BN254))
			}, fmt.Sprintf("xValues=%d/hash=%s", i, hashName))
		}
	}
}

type mulNoDependencyCircuit struct {
	X, Y     []frontend.Variable
	hashName string
}

func (c *mulNoDependencyCircuit) Define(api frontend.API) error {
	gkr := NewApi()
	var x, y constraint.GkrVariable
	var err error
	if x, err = gkr.Import(c.X); err != nil {
		return err
	}
	if y, err = gkr.Import(c.Y); err != nil {
		return err
	}
	z := gkr.Mul(x, y)
	var solution Solution
	if solution, err = gkr.Solve(api); err != nil {
		return err
	}
	X := solution.Export(x)
	Y := solution.Export(y)
	Z := solution.Export(z)
	api.Println("after solving, z=", Z, ", x=", X, ", y=", Y)

	for i := range c.X {
		api.Println("z@", i, " = ", Z[i])
		api.Println("x.y = ", api.Mul(c.X[i], c.Y[i]))
		api.AssertIsEqual(Z[i], api.Mul(c.X[i], c.Y[i]))
	}

	return solution.Verify(c.hashName)
}

func TestMulNoDependency(t *testing.T) {
	assert := test.NewAssert(t)
	xValuess := [][]frontend.Variable{
		{1, 2},
	}
	yValuess := [][]frontend.Variable{
		{0, 3},
	}

	hashes := []string{"-1", "-20"}

	for i := range xValuess {
		for _, hashName := range hashes {

			assignment := mulNoDependencyCircuit{
				X: xValuess[i],
				Y: yValuess[i],
			}
			circuit := mulNoDependencyCircuit{
				X:        make([]frontend.Variable, len(xValuess[i])),
				Y:        make([]frontend.Variable, len(yValuess[i])),
				hashName: hashName,
			}
			assert.Run(func(assert *test.Assert) {
				assert.CheckCircuit(&circuit, test.WithValidAssignment(&assignment), test.WithCurves(ecc.BN254))
			}, fmt.Sprintf("xValues=%d/hash=%s", i, hashName))
		}
	}
}

type mulWithDependencyCircuit struct {
	XLast    frontend.Variable
	Y        []frontend.Variable
	hashName string
}

func (c *mulWithDependencyCircuit) Define(api frontend.API) error {
	gkr := NewApi()
	var x, y constraint.GkrVariable
	var err error

	X := make([]frontend.Variable, len(c.Y))
	X[len(c.Y)-1] = c.XLast
	if x, err = gkr.Import(X); err != nil {
		return err
	}
	if y, err = gkr.Import(c.Y); err != nil {
		return err
	}
	z := gkr.Mul(x, y)

	for i := len(X) - 1; i > 0; i-- {
		gkr.Series(x, z, i-1, i)
	}

	var solution Solution
	if solution, err = gkr.Solve(api); err != nil {
		return err
	}
	X = solution.Export(x)
	Y := solution.Export(y)
	Z := solution.Export(z)

	api.Println("after solving, z=", Z, ", x=", X, ", y=", Y)

	lastI := len(X) - 1
	api.AssertIsEqual(Z[lastI], api.Mul(c.XLast, Y[lastI]))
	for i := 0; i < lastI; i++ {
		api.AssertIsEqual(Z[i], api.Mul(Z[i+1], Y[i]))
	}
	return solution.Verify(c.hashName)
}

func TestSolveMulWithDependency(t *testing.T) {
	assert := test.NewAssert(t)
	assignment := mulWithDependencyCircuit{
		XLast: 1,
		Y:     []frontend.Variable{3, 2},
	}
	circuit := mulWithDependencyCircuit{Y: make([]frontend.Variable, len(assignment.Y)), hashName: "-20"}
	assert.CheckCircuit(&circuit, test.WithValidAssignment(&assignment), test.WithCurves(ecc.BN254))
}

func TestApiMul(t *testing.T) {
	var (
		x   constraint.GkrVariable
		y   constraint.GkrVariable
		z   constraint.GkrVariable
		err error
	)
	api := NewApi()
	x, err = api.Import([]frontend.Variable{nil, nil})
	require.NoError(t, err)
	y, err = api.Import([]frontend.Variable{nil, nil})
	require.NoError(t, err)
	z = api.Mul(x, y)
	assertSliceEqual(t, api.toStore.Circuit[z].Inputs, []int{int(x), int(y)}) // TODO: Find out why assert.Equal gives false positives ( []*Wire{x,x} as second argument passes when it shouldn't )
}

func BenchmarkMiMCMerkleTree(b *testing.B) {
	depth := 14
	bottom := make([]frontend.Variable, 1<<depth)

	for i := 0; i < 1<<depth; i++ {
		bottom[i] = i
	}

	assignment := benchMiMCMerkleTreeCircuit{
		depth:   depth,
		XBottom: bottom[:len(bottom)/2],
		YBottom: bottom[len(bottom)/2:],
	}

	circuit := benchMiMCMerkleTreeCircuit{
		depth:   depth,
		XBottom: make([]frontend.Variable, len(assignment.XBottom)),
		YBottom: make([]frontend.Variable, len(assignment.YBottom)),
	}

	benchProof(b, &circuit, &assignment)
}

func benchCompile(b *testing.B, circuit frontend.Circuit) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit, frontend.WithCompressThreshold(compressThreshold))
		require.NoError(b, err)
	}
}

func benchProof(b *testing.B, circuit, assignment frontend.Circuit) {
	fmt.Println("compiling...")
	start := time.Now().UnixMicro()
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit, frontend.WithCompressThreshold(compressThreshold))
	require.NoError(b, err)
	fmt.Println("compiled in", time.Now().UnixMicro()-start, "μs")
	fullWitness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
	require.NoError(b, err)
	//publicWitness := fullWitness.Public()
	fmt.Println("setting up...")
	pk, _, err := groth16.Setup(cs)
	require.NoError(b, err)

	fmt.Println("solving and proving...")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		id := rand.Uint32() % 256 //#nosec G404 -- This is a false positive
		start = time.Now().UnixMicro()
		fmt.Println("groth16 proving", id)
		_, err = groth16.Prove(cs, pk, fullWitness)
		require.NoError(b, err)
		fmt.Println("groth16 proved", id, "in", time.Now().UnixMicro()-start, "μs")

		fmt.Println("mimc total calls: fr=", mimcFrTotalCalls, ", snark=", mimcSnarkTotalCalls)
	}
}

type benchMiMCMerkleTreeCircuit struct {
	depth   int
	XBottom []frontend.Variable
	YBottom []frontend.Variable
}

// hard-coded bn254
func (c *benchMiMCMerkleTreeCircuit) Define(api frontend.API) error {

	X := make([]frontend.Variable, 2*len(c.XBottom))
	Y := make([]frontend.Variable, 2*len(c.YBottom))

	copy(X, c.XBottom)
	copy(Y, c.YBottom)

	X[len(X)-1] = 0
	Y[len(X)-1] = 0

	var x, y constraint.GkrVariable
	var err error

	gkr := NewApi()
	if x, err = gkr.Import(X); err != nil {
		return err
	}
	if y, err = gkr.Import(Y); err != nil {
		return err
	}

	// cheat{
	gkr.toStore.Circuit = append(gkr.toStore.Circuit, constraint.GkrWire{
		Gate:   "mimc",
		Inputs: []int{int(x), int(y)},
	})
	gkr.assignments = append(gkr.assignments, nil)
	z := constraint.GkrVariable(2)
	// }

	offset := 1 << (c.depth - 1)
	for d := c.depth - 2; d >= 0; d-- {
		for i := 0; i < 1<<d; i++ {
			gkr.Series(x, z, offset+i, offset-1-2*i)
			gkr.Series(y, z, offset+i, offset-2-2*i)
		}
		offset += 1 << d
	}

	solution, err := gkr.Solve(api)
	if err != nil {
		return err
	}
	Z := solution.Export(z)

	challenge, err := api.(frontend.Committer).Commit(Z...)
	if err != nil {
		return err
	}

	return solution.Verify("-20", challenge)
}

func registerMiMC() {
	bn254.RegisterHashBuilder("mimc", gcHash.MIMC_BN254.New)
	stdHash.Register("mimc", func(api frontend.API) (stdHash.FieldHasher, error) {
		m, err := mimc.NewMiMC(api)
		return &m, err
	})
}

func registerConstant(c int) {
	name := strconv.Itoa(c)
	bls12377.RegisterHashBuilder(name, func() hash.Hash {
		return bls12377.ConstPseudoHash(c)
	})

	bls12381.RegisterHashBuilder(name, func() hash.Hash {
		return bls12381.ConstPseudoHash(c)
	})
	bls24315.RegisterHashBuilder(name, func() hash.Hash {
		return bls24315.ConstPseudoHash(c)
	})
	bls24317.RegisterHashBuilder(name, func() hash.Hash {
		return bls24317.ConstPseudoHash(c)
	})
	bn254.RegisterHashBuilder(name, func() hash.Hash {
		return bn254.ConstPseudoHash(c)
	})
	bw6633.RegisterHashBuilder(name, func() hash.Hash {
		return bw6633.ConstPseudoHash(c)
	})
	bw6761.RegisterHashBuilder(name, func() hash.Hash {
		return bw6761.ConstPseudoHash(c)
	})

	stdHash.Register(name, func(frontend.API) (stdHash.FieldHasher, error) {
		return constPseudoHash(c), nil
	})
}

func init() {
	registerMiMC()
	registerConstant(-1)
	registerConstant(-20)
	registerMiMCGate()
}

func registerMiMCGate() {
	// register mimc gate
	panicIfError(RegisterGate("mimc", func(api GateAPI, input ...frontend.Variable) frontend.Variable {
		mimcSnarkTotalCalls++

		if len(input) != 2 {
			panic("mimc has fan-in 2")
		}
		sum := api.Add(input[0], input[1] /*, m.Ark*/)

		sumCubed := api.Mul(sum, sum, sum) // sum^3
		return api.Mul(sumCubed, sumCubed, sum)
	}, 2, WithDegree(7)))

	// register fr version of mimc gate
	panicIfError(gkr.RegisterGate("mimc", func(input ...fr.Element) (res fr.Element) {
		var sum fr.Element

		sum.
			Add(&input[0], &input[1]) //.Add(&sum, &m.ark)

		res.Square(&sum)    // sum^2
		res.Mul(&res, &sum) // sum^3
		res.Square(&res)    //sum^6
		res.Mul(&res, &sum) //sum^7

		mimcFrTotalCalls++
		return res
	}, 2, gkr.WithDegree(7)))
}

type constPseudoHash int

func (c constPseudoHash) Sum() frontend.Variable {
	return int(c)
}

func (c constPseudoHash) Write(...frontend.Variable) {}

func (c constPseudoHash) Reset() {}

var mimcFrTotalCalls = 0

type mimcNoGkrCircuit struct {
	X         []frontend.Variable
	Y         []frontend.Variable
	mimcDepth int
}

func (c *mimcNoGkrCircuit) Define(api frontend.API) error {
	Z := make([]frontend.Variable, len(c.X))
	zSum := frontend.Variable(0)
	for i := range Z {
		Z[i] = c.Y[i]
		for j := 0; j < c.mimcDepth; j++ {
			Z[i] = MiMCCipherGate{Ark: 0}.Evaluate(api, c.X[i], Z[i])
		}
		zSum = api.Add(zSum, Z[i])
	}
	api.AssertIsDifferent(zSum, 0)
	return nil
}

func BenchmarkMiMCMerkleTreeNoGkrNoDep(b *testing.B) {
	nbInstances := 1 << 18
	X := make([]frontend.Variable, nbInstances)
	Y := make([]frontend.Variable, nbInstances)
	for i := range X {
		X[i] = i
		Y[i] = -2*i + 1
	}
	assignment := mimcNoGkrCircuit{
		X: X,
		Y: Y,
	}
	circuit := mimcNoGkrCircuit{
		X: make([]frontend.Variable, nbInstances),
		Y: make([]frontend.Variable, nbInstances),
	}

	benchProof(b, &circuit, &assignment)
}

type mimcNoDepCircuit struct {
	X         []frontend.Variable
	Y         []frontend.Variable
	mimcDepth int
	hashName  string
}

func (c *mimcNoDepCircuit) Define(api frontend.API) error {
	_gkr := NewApi()
	x, err := _gkr.Import(c.X)
	if err != nil {
		return err
	}
	var (
		y, z     constraint.GkrVariable
		solution Solution
	)
	if y, err = _gkr.Import(c.Y); err != nil {
		return err
	}

	// cheat{
	z = y
	for i := 0; i < c.mimcDepth; i++ {
		_gkr.toStore.Circuit = append(_gkr.toStore.Circuit, constraint.GkrWire{
			Gate:   "mimc",
			Inputs: []int{int(x), int(z)},
		})
		_gkr.assignments = append(_gkr.assignments, nil)
		z = constraint.GkrVariable(len(_gkr.toStore.Circuit) - 1)
	}
	// }

	if solution, err = _gkr.Solve(api); err != nil {
		return err
	}
	Z := solution.Export(z)
	return solution.Verify(c.hashName, Z...)
}

func mimcNoDepCircuits(mimcDepth, nbInstances int, hashName string) (circuit, assignment frontend.Circuit) {
	X := make([]frontend.Variable, nbInstances)
	Y := make([]frontend.Variable, nbInstances)
	for i := range X {
		X[i] = i
		Y[i] = -2*i + 1
	}
	assignment = &mimcNoDepCircuit{
		X: X,
		Y: Y,
	}
	circuit = &mimcNoDepCircuit{
		X:         make([]frontend.Variable, nbInstances),
		Y:         make([]frontend.Variable, nbInstances),
		mimcDepth: mimcDepth,
		hashName:  hashName,
	}
	return
}

func BenchmarkMiMCNoDepSolve(b *testing.B) {
	//defer profile.Start().Stop()
	circuit, assignment := mimcNoDepCircuits(1, 1<<9, "-20")
	benchProof(b, circuit, assignment)
}

func BenchmarkMiMCFullDepthNoDepSolve(b *testing.B) {
	circuit, assignment := mimcNoDepCircuits(91, 1<<19, "-20")
	benchProof(b, circuit, assignment)
}

func BenchmarkMiMCFullDepthNoDepCompile(b *testing.B) {
	circuit, _ := mimcNoDepCircuits(91, 1<<17, "-20")
	benchCompile(b, circuit)
}

func BenchmarkMiMCNoGkrFullDepthSolve(b *testing.B) {
	circuit, assignment := mimcNoGkrCircuits(91, 1<<19)
	benchProof(b, circuit, assignment)
}

func TestMiMCFullDepthNoDepSolve(t *testing.T) {
	assert := test.NewAssert(t)
	for i := 0; i < 100; i++ {
		circuit, assignment := mimcNoDepCircuits(5, 1<<2, "-20")
		assert.Run(func(assert *test.Assert) {
			assert.CheckCircuit(circuit, test.WithValidAssignment(assignment), test.WithCurves(ecc.BN254))
		}, fmt.Sprintf("i=%d", i))
	}
}

func TestMiMCFullDepthNoDepSolveWithMiMCHash(t *testing.T) {
	assert := test.NewAssert(t)
	circuit, assignment := mimcNoDepCircuits(5, 1<<2, "mimc")
	assert.CheckCircuit(circuit, test.WithValidAssignment(assignment), test.WithCurves(ecc.BN254))
}

func mimcNoGkrCircuits(mimcDepth, nbInstances int) (circuit, assignment frontend.Circuit) {
	X := make([]frontend.Variable, nbInstances)
	Y := make([]frontend.Variable, nbInstances)
	for i := range X {
		X[i] = i
		Y[i] = -2*i + 1
	}
	assignment = &mimcNoGkrCircuit{
		X: X,
		Y: Y,
	}
	circuit = &mimcNoGkrCircuit{
		X:         make([]frontend.Variable, nbInstances),
		Y:         make([]frontend.Variable, nbInstances),
		mimcDepth: mimcDepth,
	}
	return
}

func TestSolveInTestEngine(t *testing.T) {
	assignment := testSolveInTestEngineCircuit{
		X: []frontend.Variable{2, 3, 4, 5, 6, 7, 8, 9},
	}
	circuit := testSolveInTestEngineCircuit{
		X: make([]frontend.Variable, len(assignment.X)),
	}

	require.NoError(t, test.IsSolved(&circuit, &assignment, ecc.BN254.ScalarField()))
	require.NoError(t, test.IsSolved(&circuit, &assignment, ecc.BLS24_315.ScalarField()))
	require.NoError(t, test.IsSolved(&circuit, &assignment, ecc.BLS12_381.ScalarField()))
	require.NoError(t, test.IsSolved(&circuit, &assignment, ecc.BLS24_317.ScalarField()))
	require.NoError(t, test.IsSolved(&circuit, &assignment, ecc.BW6_633.ScalarField()))
	require.NoError(t, test.IsSolved(&circuit, &assignment, ecc.BW6_761.ScalarField()))
	require.NoError(t, test.IsSolved(&circuit, &assignment, ecc.BLS12_377.ScalarField()))
}

type testSolveInTestEngineCircuit struct {
	X []frontend.Variable
}

func (c *testSolveInTestEngineCircuit) Define(api frontend.API) error {
	gkr := NewApi()
	x, err := gkr.Import(c.X)
	if err != nil {
		return err
	}
	Y := make([]frontend.Variable, len(c.X))
	Y[0] = 1
	y, err := gkr.Import(Y)
	if err != nil {
		return err
	}

	z := gkr.Mul(x, y)

	for i := range len(c.X) - 1 {
		gkr.Series(y, z, i+1, i)
	}

	assignments := gkr.SolveInTestEngine(api)

	product := frontend.Variable(1)
	for i := range c.X {
		api.AssertIsEqual(assignments[y][i], product)
		product = api.Mul(product, c.X[i])
		api.AssertIsEqual(assignments[z][i], product)
	}

	return nil
}
