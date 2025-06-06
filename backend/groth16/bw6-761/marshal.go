// Copyright 2020-2025 Consensys Software Inc.
// Licensed under the Apache License, Version 2.0. See the LICENSE file for details.

// Code generated by gnark DO NOT EDIT

package groth16

import (
	curve "github.com/consensys/gnark-crypto/ecc/bw6-761"

	"github.com/consensys/gnark-crypto/ecc/bw6-761/fr/pedersen"
	"github.com/consensys/gnark-crypto/utils/unsafe"
	"github.com/consensys/gnark/internal/utils"

	"fmt"
	"io"
)

// WriteTo writes binary encoding of the Proof elements to writer
// points are stored in compressed form Ar | Krs | Bs
// use WriteRawTo(...) to encode the proof without point compression
func (proof *Proof) WriteTo(w io.Writer) (n int64, err error) {
	return proof.writeTo(w, false)
}

// WriteRawTo writes binary encoding of the Proof elements to writer
// points are stored in uncompressed form Ar | Krs | Bs
// use WriteTo(...) to encode the proof with point compression
func (proof *Proof) WriteRawTo(w io.Writer) (n int64, err error) {
	return proof.writeTo(w, true)
}

func (proof *Proof) writeTo(w io.Writer, raw bool) (int64, error) {
	var enc *curve.Encoder
	if raw {
		enc = curve.NewEncoder(w, curve.RawEncoding())
	} else {
		enc = curve.NewEncoder(w)
	}

	if err := enc.Encode(&proof.Ar); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&proof.Bs); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&proof.Krs); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(proof.Commitments); err != nil {
		return enc.BytesWritten(), err
	}
	if err := enc.Encode(&proof.CommitmentPok); err != nil {
		return enc.BytesWritten(), err
	}

	return enc.BytesWritten(), nil
}

// ReadFrom attempts to decode a Proof from reader
// Proof must be encoded through WriteTo (compressed) or WriteRawTo (uncompressed)
func (proof *Proof) ReadFrom(r io.Reader) (n int64, err error) {

	dec := curve.NewDecoder(r)

	if err := dec.Decode(&proof.Ar); err != nil {
		return dec.BytesRead(), err
	}
	if err := dec.Decode(&proof.Bs); err != nil {
		return dec.BytesRead(), err
	}
	if err := dec.Decode(&proof.Krs); err != nil {
		return dec.BytesRead(), err
	}
	if err := dec.Decode(&proof.Commitments); err != nil {
		return dec.BytesRead(), err
	}
	if err := dec.Decode(&proof.CommitmentPok); err != nil {
		return dec.BytesRead(), err
	}

	return dec.BytesRead(), nil
}

// WriteTo writes binary encoding of the key elements to writer
// points are compressed
// use WriteRawTo(...) to encode the key without point compression
func (vk *VerifyingKey) WriteTo(w io.Writer) (n int64, err error) {
	return vk.writeTo(w, false)
}

// WriteRawTo writes binary encoding of the key elements to writer
// points are not compressed
// use WriteTo(...) to encode the key with point compression
func (vk *VerifyingKey) WriteRawTo(w io.Writer) (n int64, err error) {
	return vk.writeTo(w, true)
}

// writeTo serialization format:
// follows bellman format:
// https://github.com/zkcrypto/bellman/blob/fa9be45588227a8c6ec34957de3f68705f07bd92/src/groth16/mod.rs#L143
// [α]1,[β]1,[β]2,[γ]2,[δ]1,[δ]2,uint32(len(Kvk)),[Kvk]1
func (vk *VerifyingKey) writeTo(w io.Writer, raw bool) (int64, error) {
	var enc *curve.Encoder
	if raw {
		enc = curve.NewEncoder(w, curve.RawEncoding())
	} else {
		enc = curve.NewEncoder(w)
	}
	if vk.PublicAndCommitmentCommitted == nil {
		vk.PublicAndCommitmentCommitted = [][]int{} // only matters in tests
	}
	toEncode := []interface{}{
		// [α]1,[β]1,[β]2,[γ]2,[δ]1,[δ]2
		&vk.G1.Alpha,
		&vk.G1.Beta,
		&vk.G2.Beta,
		&vk.G2.Gamma,
		&vk.G1.Delta,
		&vk.G2.Delta,
		// uint32(len(Kvk)),[Kvk]1
		vk.G1.K,
		utils.IntSliceSliceToUint64SliceSlice(vk.PublicAndCommitmentCommitted),
		uint32(len(vk.CommitmentKeys)),
	}
	for _, v := range toEncode {
		if err := enc.Encode(v); err != nil {
			return enc.BytesWritten(), err
		}
	}
	var n int64
	for i := range vk.CommitmentKeys {
		var (
			m   int64
			err error
		)
		if raw {
			m, err = vk.CommitmentKeys[i].WriteRawTo(w)
		} else {
			m, err = vk.CommitmentKeys[i].WriteTo(w)
		}
		n += m
		if err != nil {
			return n + enc.BytesWritten(), err
		}
	}
	return n + enc.BytesWritten(), nil
}

// ReadFrom attempts to decode a VerifyingKey from reader
// VerifyingKey must be encoded through WriteTo (compressed) or WriteRawTo (uncompressed)
// serialization format:
// https://github.com/zkcrypto/bellman/blob/fa9be45588227a8c6ec34957de3f68705f07bd92/src/groth16/mod.rs#L143
// [α]1,[β]1,[β]2,[γ]2,[δ]1,[δ]2,uint32(len(Kvk)),[Kvk]1
func (vk *VerifyingKey) ReadFrom(r io.Reader) (int64, error) {
	return vk.readFrom(r, false)
}

// UnsafeReadFrom has the same behavior as ReadFrom, except that it will not check that decode points
// are on the curve and in the correct subgroup.
func (vk *VerifyingKey) UnsafeReadFrom(r io.Reader) (int64, error) {
	return vk.readFrom(r, true)
}

func (vk *VerifyingKey) readFrom(r io.Reader, raw bool) (int64, error) {
	var dec *curve.Decoder
	if raw {
		dec = curve.NewDecoder(r, curve.NoSubgroupChecks())
	} else {
		dec = curve.NewDecoder(r)
	}

	var publicCommitted [][]uint64
	var nbCommitments uint32

	toDecode := []interface{}{
		// [α]1,[β]1,[β]2,[γ]2,[δ]1,[δ]2
		&vk.G1.Alpha,
		&vk.G1.Beta,
		&vk.G2.Beta,
		&vk.G2.Gamma,
		&vk.G1.Delta,
		&vk.G2.Delta,
		// uint32(len(Kvk)),[Kvk]1
		&vk.G1.K,
		&publicCommitted,
		&nbCommitments,
	}

	for i, v := range toDecode {
		if err := dec.Decode(v); err != nil {
			return dec.BytesRead(), fmt.Errorf("read field %d: %w", i, err)
		}
	}

	vk.PublicAndCommitmentCommitted = utils.Uint64SliceSliceToIntSliceSlice(publicCommitted)

	var n int64
	for i := 0; i < int(nbCommitments); i++ {
		var (
			m   int64
			err error
		)
		commitmentKey := pedersen.VerifyingKey{}
		if raw {
			m, err = commitmentKey.UnsafeReadFrom(r)
		} else {
			m, err = commitmentKey.ReadFrom(r)
		}
		n += m
		if err != nil {
			return n + dec.BytesRead(), fmt.Errorf("read commitment key %d: %w", i, err)
		}
		vk.CommitmentKeys = append(vk.CommitmentKeys, commitmentKey)
	}
	if len(vk.CommitmentKeys) != int(nbCommitments) {
		return n + dec.BytesRead(), fmt.Errorf("invalid number of commitment keys. Expected %d got %d", nbCommitments, len(vk.CommitmentKeys))
	}

	// recompute vk.e (e(α, β)) and  -[δ]2, -[γ]2
	if err := vk.Precompute(); err != nil {
		return n + dec.BytesRead(), fmt.Errorf("precompute: %w", err)
	}

	return n + dec.BytesRead(), nil
}

// WriteTo writes binary encoding of the key elements to writer
// points are compressed
// use WriteRawTo(...) to encode the key without point compression
func (pk *ProvingKey) WriteTo(w io.Writer) (n int64, err error) {
	return pk.writeTo(w, false)
}

// WriteRawTo writes binary encoding of the key elements to writer
// points are not compressed
// use WriteTo(...) to encode the key with point compression
func (pk *ProvingKey) WriteRawTo(w io.Writer) (n int64, err error) {
	return pk.writeTo(w, true)
}

func (pk *ProvingKey) writeTo(w io.Writer, raw bool) (int64, error) {
	n, err := pk.Domain.WriteTo(w)
	if err != nil {
		return n, err
	}

	var enc *curve.Encoder
	if raw {
		enc = curve.NewEncoder(w, curve.RawEncoding())
	} else {
		enc = curve.NewEncoder(w)
	}
	nbWires := uint64(len(pk.InfinityA))

	toEncode := []interface{}{
		&pk.G1.Alpha,
		&pk.G1.Beta,
		&pk.G1.Delta,
		pk.G1.A,
		pk.G1.B,
		pk.G1.Z,
		pk.G1.K,
		&pk.G2.Beta,
		&pk.G2.Delta,
		pk.G2.B,
		nbWires,
		pk.NbInfinityA,
		pk.NbInfinityB,
		pk.InfinityA,
		pk.InfinityB,
		uint32(len(pk.CommitmentKeys)),
	}

	for _, v := range toEncode {
		if err := enc.Encode(v); err != nil {
			return n + enc.BytesWritten(), err
		}
	}

	for i := range pk.CommitmentKeys {
		var (
			n2  int64
			err error
		)
		if raw {
			n2, err = pk.CommitmentKeys[i].WriteRawTo(w)
		} else {
			n2, err = pk.CommitmentKeys[i].WriteTo(w)
		}

		n += n2
		if err != nil {
			return n + enc.BytesWritten(), err
		}
	}

	return n + enc.BytesWritten(), nil

}

// ReadFrom attempts to decode a ProvingKey from reader
// ProvingKey must be encoded through WriteTo (compressed) or WriteRawTo (uncompressed)
// note that we don't check that the points are on the curve or in the correct subgroup at this point
func (pk *ProvingKey) ReadFrom(r io.Reader) (int64, error) {
	return pk.readFrom(r)
}

// UnsafeReadFrom behaves like ReadFrom excepts it doesn't check if the decoded points are on the curve
// or in the correct subgroup
func (pk *ProvingKey) UnsafeReadFrom(r io.Reader) (int64, error) {
	return pk.readFrom(r, curve.NoSubgroupChecks())
}

func (pk *ProvingKey) readFrom(r io.Reader, decOptions ...func(*curve.Decoder)) (int64, error) {
	n, err := pk.Domain.ReadFrom(r)
	if err != nil {
		return n, fmt.Errorf("read domain: %w", err)
	}

	dec := curve.NewDecoder(r, decOptions...)

	var nbWires uint64
	var nbCommitments uint32

	toDecode := []interface{}{
		&pk.G1.Alpha,
		&pk.G1.Beta,
		&pk.G1.Delta,
		&pk.G1.A,
		&pk.G1.B,
		&pk.G1.Z,
		&pk.G1.K,
		&pk.G2.Beta,
		&pk.G2.Delta,
		&pk.G2.B,
		&nbWires,
		&pk.NbInfinityA,
		&pk.NbInfinityB,
	}

	for i, v := range toDecode {
		if err := dec.Decode(v); err != nil {
			return n + dec.BytesRead(), fmt.Errorf("read field %d: %w", i, err)
		}
	}
	pk.InfinityA = make([]bool, nbWires)
	pk.InfinityB = make([]bool, nbWires)

	if err := dec.Decode(&pk.InfinityA); err != nil {
		return n + dec.BytesRead(), fmt.Errorf("read InfinityA: %w", err)
	}
	if err := dec.Decode(&pk.InfinityB); err != nil {
		return n + dec.BytesRead(), fmt.Errorf("read InfinityB: %w", err)
	}
	if err := dec.Decode(&nbCommitments); err != nil {
		return n + dec.BytesRead(), fmt.Errorf("read nbCommitments: %w", err)
	}
	for i := 0; i < int(nbCommitments); i++ {
		cpkey := pedersen.ProvingKey{}
		n2, err := cpkey.ReadFrom(r)
		n += n2
		if err != nil {
			return n + dec.BytesRead(), fmt.Errorf("read commitment key %d: %w", i, err)
		}
		pk.CommitmentKeys = append(pk.CommitmentKeys, cpkey)
	}
	if len(pk.CommitmentKeys) != int(nbCommitments) {
		return n + dec.BytesRead(), fmt.Errorf("invalid number of commitment keys. Expected %d got %d", nbCommitments, len(pk.CommitmentKeys))
	}

	return n + dec.BytesRead(), nil
}

// WriteDump behaves like WriteRawTo, excepts, the slices of points are "dumped" using gnark-crypto/utils/unsafe
// Output is compatible with ReadDump, with the caveat that, not only the points are not checked for
// correctness, but the raw bytes are platform dependent (endianness, etc.)
func (pk *ProvingKey) WriteDump(w io.Writer) error {
	// it behaves like WriteRawTo, excepts, the slices of points are "dumped" using gnark-crypto/utils/unsafe

	// start by writing an unsafe marker to fail early.
	if err := unsafe.WriteMarker(w); err != nil {
		return err
	}

	if _, err := pk.Domain.WriteTo(w); err != nil {
		return err
	}

	enc := curve.NewEncoder(w, curve.RawEncoding())
	nbWires := uint64(len(pk.InfinityA))

	toEncode := []interface{}{
		&pk.G1.Alpha,
		&pk.G1.Beta,
		&pk.G1.Delta,
		// pk.G1.A,
		// pk.G1.B,
		// pk.G1.Z,
		// pk.G1.K,
		&pk.G2.Beta,
		&pk.G2.Delta,
		// pk.G2.B,
		nbWires,
		pk.NbInfinityA,
		pk.NbInfinityB,
		pk.InfinityA,
		pk.InfinityB,
		uint32(len(pk.CommitmentKeys)),
	}

	for _, v := range toEncode {
		if err := enc.Encode(v); err != nil {
			return err
		}
	}

	// dump slices of points
	if err := unsafe.WriteSlice(w, pk.G1.A); err != nil {
		return err
	}
	if err := unsafe.WriteSlice(w, pk.G1.B); err != nil {
		return err
	}
	if err := unsafe.WriteSlice(w, pk.G1.Z); err != nil {
		return err
	}
	if err := unsafe.WriteSlice(w, pk.G1.K); err != nil {
		return err
	}
	if err := unsafe.WriteSlice(w, pk.G2.B); err != nil {
		return err
	}

	for i := range pk.CommitmentKeys {
		if err := unsafe.WriteSlice(w, pk.CommitmentKeys[i].Basis); err != nil {
			return err
		}
		if err := unsafe.WriteSlice(w, pk.CommitmentKeys[i].BasisExpSigma); err != nil {
			return err
		}
	}

	return nil
}

// ReadDump reads a ProvingKey from a dump written by WriteDump.
// This is platform dependent and very unsafe (no checks, no endianness translation, etc.)
func (pk *ProvingKey) ReadDump(r io.Reader) error {
	// read the marker to fail early in case of malformed input
	if err := unsafe.ReadMarker(r); err != nil {
		return fmt.Errorf("read marker: %w", err)
	}

	if _, err := pk.Domain.ReadFrom(r); err != nil {
		return fmt.Errorf("read domain: %w", err)
	}

	dec := curve.NewDecoder(r, curve.NoSubgroupChecks())

	var nbWires uint64
	var nbCommitments uint32

	toDecode := []interface{}{
		&pk.G1.Alpha,
		&pk.G1.Beta,
		&pk.G1.Delta,
		// &pk.G1.A,
		// &pk.G1.B,
		// &pk.G1.Z,
		// &pk.G1.K,
		&pk.G2.Beta,
		&pk.G2.Delta,
		// &pk.G2.B,
		&nbWires,
		&pk.NbInfinityA,
		&pk.NbInfinityB,
	}

	for i, v := range toDecode {
		if err := dec.Decode(v); err != nil {
			return fmt.Errorf("read field %d: %w", i, err)
		}
	}
	pk.InfinityA = make([]bool, nbWires)
	pk.InfinityB = make([]bool, nbWires)

	if err := dec.Decode(&pk.InfinityA); err != nil {
		return fmt.Errorf("read InfinityA: %w", err)
	}
	if err := dec.Decode(&pk.InfinityB); err != nil {
		return fmt.Errorf("read InfinityB: %w", err)
	}
	if err := dec.Decode(&nbCommitments); err != nil {
		return fmt.Errorf("read nbCommitments: %w", err)
	}

	// read slices of points
	var err error
	pk.G1.A, _, err = unsafe.ReadSlice[[]curve.G1Affine](r)
	if err != nil {
		return fmt.Errorf("read G1.A: %w", err)
	}
	pk.G1.B, _, err = unsafe.ReadSlice[[]curve.G1Affine](r)
	if err != nil {
		return fmt.Errorf("read G1.B: %w", err)
	}
	pk.G1.Z, _, err = unsafe.ReadSlice[[]curve.G1Affine](r)
	if err != nil {
		return fmt.Errorf("read G1.Z: %w", err)
	}
	pk.G1.K, _, err = unsafe.ReadSlice[[]curve.G1Affine](r)
	if err != nil {
		return fmt.Errorf("read G1.K: %w", err)
	}
	pk.G2.B, _, err = unsafe.ReadSlice[[]curve.G2Affine](r)
	if err != nil {
		return fmt.Errorf("read G2.B: %w", err)
	}

	for i := 0; i < int(nbCommitments); i++ {
		cpkey := pedersen.ProvingKey{}
		cpkey.Basis, _, err = unsafe.ReadSlice[[]curve.G1Affine](r)
		if err != nil {
			return fmt.Errorf("read commitment basis %d: %w", i, err)
		}
		cpkey.BasisExpSigma, _, err = unsafe.ReadSlice[[]curve.G1Affine](r)
		if err != nil {
			return fmt.Errorf("read commitment basisExpSigma %d: %w", i, err)
		}
		pk.CommitmentKeys = append(pk.CommitmentKeys, cpkey)
	}
	if len(pk.CommitmentKeys) != int(nbCommitments) {
		return fmt.Errorf("invalid number of commitment keys. Expected %d got %d", nbCommitments, len(pk.CommitmentKeys))
	}

	return nil

}
