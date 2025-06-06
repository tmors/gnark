// Copyright 2020-2025 Consensys Software Inc.
// Licensed under the Apache License, Version 2.0. See the LICENSE file for details.

// Code generated by gnark DO NOT EDIT

package mpcsetup

import (
	curve "github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/fft"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/pedersen"
	"github.com/consensys/gnark-crypto/ecc/bn254/mpcsetup"
	"github.com/consensys/gnark/backend/groth16"
	groth16Impl "github.com/consensys/gnark/backend/groth16/bn254"
)

// Seal performs the final contribution and outputs the proving and verifying keys.
// No randomization is performed at this step.
// A verifier should simply re-run this and check
// that it produces the same values.
// beaconChallenge is a random beacon of moderate entropy evaluated at a time later than the latest contribution.
// It seeds a final "contribution" to the protocol, reproducible by any verifier.
// For more information on random beacons, refer to https://a16zcrypto.com/posts/article/public-randomness-and-randomness-beacons/
// Organizations such as the League of Entropy (https://leagueofentropy.com/) provide such beacons. THIS IS NOT A RECOMMENDATION OR ENDORSEMENT.
// WARNING: Seal modifies p, just as Contribute does.
// The result will be an INVALID Phase1 object, since no proof of correctness is produced.
func (p *Phase2) Seal(commons *SrsCommons, evals *Phase2Evaluations, beaconChallenge []byte) (groth16.ProvingKey, groth16.VerifyingKey) {

	// final contributions
	contributions := mpcsetup.BeaconContributions(p.hash(), []byte("Groth16 MPC Setup - Phase2"), beaconChallenge, 1+len(p.Sigmas))
	p.update(&contributions[0], contributions[1:])

	_, _, _, g2 := curve.Generators()

	var (
		pk groth16Impl.ProvingKey
		vk groth16Impl.VerifyingKey
	)

	// Initialize PK
	pk.Domain = *fft.NewDomain(uint64(len(commons.G1.AlphaTau)))
	pk.G1.Alpha.Set(&commons.G1.AlphaTau[0])
	pk.G1.Beta.Set(&commons.G1.BetaTau[0])
	pk.G1.Delta.Set(&p.Parameters.G1.Delta)
	pk.G1.Z = p.Parameters.G1.Z
	bitReverse(pk.G1.Z)

	pk.G1.K = p.Parameters.G1.PKK
	pk.G2.Beta.Set(&commons.G2.Beta)
	pk.G2.Delta.Set(&p.Parameters.G2.Delta)

	// Filter out infinity points
	nWires := len(evals.G1.A)
	pk.InfinityA = make([]bool, nWires)
	A := make([]curve.G1Affine, nWires)
	j := 0
	for i, e := range evals.G1.A {
		if e.IsInfinity() {
			pk.InfinityA[i] = true
			continue
		}
		A[j] = evals.G1.A[i]
		j++
	}
	pk.G1.A = A[:j]
	pk.NbInfinityA = uint64(nWires - j)

	pk.InfinityB = make([]bool, nWires)
	B := make([]curve.G1Affine, nWires)
	j = 0
	for i, e := range evals.G1.B {
		if e.IsInfinity() {
			pk.InfinityB[i] = true
			continue
		}
		B[j] = evals.G1.B[i]
		j++
	}
	pk.G1.B = B[:j]
	pk.NbInfinityB = uint64(nWires - j)

	B2 := make([]curve.G2Affine, nWires)
	j = 0
	for i, e := range evals.G2.B {
		if e.IsInfinity() {
			// pk.InfinityB[i] = true should be the same as in B
			continue
		}
		B2[j] = evals.G2.B[i]
		j++
	}
	pk.G2.B = B2[:j]

	// Initialize VK
	vk.G1.Alpha.Set(&commons.G1.AlphaTau[0])
	vk.G1.Beta.Set(&commons.G1.BetaTau[0])
	vk.G1.Delta.Set(&p.Parameters.G1.Delta)
	vk.G2.Beta.Set(&commons.G2.Beta)
	vk.G2.Delta.Set(&p.Parameters.G2.Delta)
	vk.G2.Gamma.Set(&g2)
	vk.G1.K = evals.G1.VKK

	vk.CommitmentKeys = make([]pedersen.VerifyingKey, len(evals.G1.CKK))
	pk.CommitmentKeys = make([]pedersen.ProvingKey, len(evals.G1.CKK))
	for i := range vk.CommitmentKeys {
		vk.CommitmentKeys[i].G = g2
		vk.CommitmentKeys[i].GSigmaNeg.Neg(&p.Parameters.G2.Sigma[i])

		pk.CommitmentKeys[i].Basis = evals.G1.CKK[i]
		pk.CommitmentKeys[i].BasisExpSigma = p.Parameters.G1.SigmaCKK[i]
	}
	vk.PublicAndCommitmentCommitted = evals.PublicAndCommitmentCommitted

	// sets e, -[δ]2, -[γ]2
	if err := vk.Precompute(); err != nil {
		panic(err)
	}

	return &pk, &vk
}
