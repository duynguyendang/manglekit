package graph

import (
	"encoding/binary"
	"fmt"
)

// Prefix keys for BadgerDB to route to proper index tables.
const (
	PrefixSPOg byte = 0x10 // Subject-Predicate-Object-Graph (Standard Forward)
	PrefixPOSg byte = 0x11 // Predicate-Object-Subject-Graph (Reverse Lookup)
	PrefixPSOg byte = 0x12 // Predicate-Subject-Object-Graph (Graph Filtering)
	PrefixGSPO byte = 0x13 // Graph-Subject-Predicate-Object (Isolation Layer)

	PrefixSystem byte = 0xFF // Metadata (Count, Schema bounds, Leiden Communities)
)

var (
	KeyFactCount = []byte{PrefixSystem, 0x01} // uint64 count of stored facts
)

// EncodeQuadKey produces a strictly aligned 33-byte slice for lexicographical sorting.
// By using Big-Endian representation, the numeric scale of uint64 perfectly
// aligns with the byte sequence scale inside BadgerDB SSTables, enabling
// optimal binary searches via LFTJ (Leapfrog Triejoin).
func EncodeQuadKey(prefix byte, s, p, o, g uint64) []byte {
	key := make([]byte, 33) // 1 byte prefix + 4 * 8 byte uint64s
	key[0] = prefix

	switch prefix {
	case PrefixSPOg:
		binary.BigEndian.PutUint64(key[1:9], s)
		binary.BigEndian.PutUint64(key[9:17], p)
		binary.BigEndian.PutUint64(key[17:25], o)
		binary.BigEndian.PutUint64(key[25:33], g)
	case PrefixPOSg:
		// Used to rapidly find subjects that share the same Predicate/Object relation (e.g. Reverse Index)
		binary.BigEndian.PutUint64(key[1:9], p)
		binary.BigEndian.PutUint64(key[9:17], o)
		binary.BigEndian.PutUint64(key[17:25], s)
		binary.BigEndian.PutUint64(key[25:33], g)
	case PrefixPSOg:
		binary.BigEndian.PutUint64(key[1:9], p)
		binary.BigEndian.PutUint64(key[9:17], s)
		binary.BigEndian.PutUint64(key[17:25], o)
		binary.BigEndian.PutUint64(key[25:33], g)
	case PrefixGSPO:
		// Used to rapidly filter everything inside a tenant sandbox or epoch context
		binary.BigEndian.PutUint64(key[1:9], g)
		binary.BigEndian.PutUint64(key[9:17], s)
		binary.BigEndian.PutUint64(key[17:25], p)
		binary.BigEndian.PutUint64(key[25:33], o)
	default:
		panic(fmt.Sprintf("EncodeQuadKey: Invalid index routing prefix 0x%X", prefix))
	}
	return key
}

// DecodeQuadKey reads the bytes back out of the index, correctly reversing the permutation.
func DecodeQuadKey(key []byte) (s, p, o, g uint64, err error) {
	if len(key) != 33 {
		return 0, 0, 0, 0, fmt.Errorf("invalid quad key length: got %d, want 33", len(key))
	}

	prefix := key[0]

	switch prefix {
	case PrefixSPOg:
		s = binary.BigEndian.Uint64(key[1:9])
		p = binary.BigEndian.Uint64(key[9:17])
		o = binary.BigEndian.Uint64(key[17:25])
		g = binary.BigEndian.Uint64(key[25:33])
	case PrefixPOSg:
		p = binary.BigEndian.Uint64(key[1:9])
		o = binary.BigEndian.Uint64(key[9:17])
		s = binary.BigEndian.Uint64(key[17:25])
		g = binary.BigEndian.Uint64(key[25:33])
	case PrefixPSOg:
		p = binary.BigEndian.Uint64(key[1:9])
		s = binary.BigEndian.Uint64(key[9:17])
		o = binary.BigEndian.Uint64(key[17:25])
		g = binary.BigEndian.Uint64(key[25:33])
	case PrefixGSPO:
		g = binary.BigEndian.Uint64(key[1:9])
		s = binary.BigEndian.Uint64(key[9:17])
		p = binary.BigEndian.Uint64(key[17:25])
		o = binary.BigEndian.Uint64(key[25:33])
	default:
		return 0, 0, 0, 0, fmt.Errorf("invalid quad prefix: 0x%X", prefix)
	}
	return s, p, o, g, nil
}
