package difficulty

import (
	"math/big"
	"reflect"
	"strconv"
	"testing"

	"github.com/bytom/consensus"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// A lower difficulty Int actually reflects a more difficult mining progress.
func TestCalcNextRequiredDifficulty(t *testing.T) {
	targetTimeSpan := uint64(consensus.BlocksPerRetarget * consensus.TargetSecondsPerBlock)
	cases := []struct {
		lastBH    *types.BlockHeader
		compareBH *types.BlockHeader
		want      uint64
	}{
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan,
				Bits:      BigToCompact(big.NewInt(1000))},
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0},
			BigToCompact(big.NewInt(1000)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan * 2,
				Bits:      BigToCompact(big.NewInt(1000))},
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0},
			BigToCompact(big.NewInt(2000)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget - 1,
				Timestamp: targetTimeSpan*2 - consensus.TargetSecondsPerBlock,
				Bits:      BigToCompact(big.NewInt(1000))},
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0},
			BigToCompact(big.NewInt(1000)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan / 2,
				Bits:      BigToCompact(big.NewInt(1000))},
			&types.BlockHeader{
				Height:    0,
				Timestamp: 0},
			BigToCompact(big.NewInt(500)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget * 2,
				Timestamp: targetTimeSpan + targetTimeSpan*2,
				Bits:      BigToCompact(big.NewInt(1000))},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan},
			BigToCompact(big.NewInt(2000)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget * 2,
				Timestamp: targetTimeSpan + targetTimeSpan/2,
				Bits:      BigToCompact(big.NewInt(1000))},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan},
			BigToCompact(big.NewInt(500)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget*2 - 1,
				Timestamp: targetTimeSpan + targetTimeSpan*2 - consensus.TargetSecondsPerBlock,
				Bits:      BigToCompact(big.NewInt(1000))},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan},
			BigToCompact(big.NewInt(1000)),
		},
		{
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget*2 - 1,
				Timestamp: targetTimeSpan + targetTimeSpan/2 - consensus.TargetSecondsPerBlock,
				Bits:      BigToCompact(big.NewInt(1000))},
			&types.BlockHeader{
				Height:    consensus.BlocksPerRetarget,
				Timestamp: targetTimeSpan},
			BigToCompact(big.NewInt(1000)),
		},
	}

	for i, c := range cases {
		if got := CalcNextRequiredDifficulty(c.lastBH, c.compareBH); got != c.want {
			t.Errorf("Compile(%d) = %d want %d\n", i, got, c.want)
			return
		}
	}
}

func TestHashToBig(t *testing.T) {
	cases := []struct {
		in  [32]byte
		out [32]byte
	}{
		{
			in: [32]byte{
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
			},
			out: [32]byte{
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
				0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
			},
		},
		{
			in: [32]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
				0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
			},
			out: [32]byte{
				0x0f, 0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09, 0x08,
				0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			in: [32]byte{
				0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7,
				0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xef,
				0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7,
				0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff,
			},
			out: [32]byte{
				0xff, 0xfe, 0xfd, 0xfc, 0xfb, 0xfa, 0xf9, 0xf8,
				0xf7, 0xf6, 0xf5, 0xf4, 0xf3, 0xf2, 0xf1, 0xf0,
				0xef, 0xee, 0xed, 0xec, 0xeb, 0xea, 0xe9, 0xe8,
				0xe7, 0xe6, 0xe5, 0xe4, 0xe3, 0xe2, 0xe1, 0xe0,
			},
		},
	}

	for i, c := range cases {
		bhash := bc.NewHash(c.in)
		result := HashToBig(&bhash).Bytes()

		var resArr [32]byte
		copy(resArr[:], result)

		if !reflect.DeepEqual(resArr, c.out) {
			t.Errorf("TestHashToBig test #%d failed:\n\tgot\t%x\n\twant\t%x\n", i, resArr, c.out)
			return
		}
	}
}

func TestCompactToBig(t *testing.T) {
	cases := []struct {
		in  string
		out *big.Int
	}{
		{
			in: `00000000` + //Exponent
				`0` + //Sign
				`0000000000000000000000000000000000000000000000000000000`, //Mantissa
			out: big.NewInt(0),
		},
		{
			in: `00000000` + //Exponent
				`1` + //Sign
				`0000000000000000000000000000000000000000000000000000000`, //Mantissa
			out: big.NewInt(0),
		},
		{
			in: `00000001` + //Exponent
				`0` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(1),
		},
		{
			in: `00000001` + //Exponent
				`1` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(-1),
		},
		{
			in: `00000011` + //Exponent
				`0` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(65536),
		},
		{
			in: `00000011` + //Exponent
				`1` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(-65536),
		},
		{
			in: `00000100` + //Exponent
				`0` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(16777216),
		},
		{
			in: `00000100` + //Exponent
				`1` + //Sign
				`0000000000000000000000000000000000000010000000000000000`, //Mantissa
			out: big.NewInt(-16777216),
		},
		{
			//btm PowMin test
			// PowMinBits = 2161727821138738707, i.e 0x1e000000000dbe13, as defined
			// in /consensus/general.go
			in: `00011110` + //Exponent
				`0` + //Sign
				`0000000000000000000000000000000000011011011111000010011`, //Mantissa
			out: big.NewInt(0).Lsh(big.NewInt(0x0dbe13), 27*8), //2161727821138738707
		},
	}

	for i, c := range cases {
		compact, _ := strconv.ParseUint(c.in, 2, 64)
		r := CompactToBig(compact)
		if r.Cmp(c.out) != 0 {
			t.Error("TestCompactToBig test #", i, "failed: got", r, "want", c.out)
			return
		}
	}
}

func TestBigToCompact(t *testing.T) {
	// basic tests
	tests := []struct {
		in  int64
		out uint64
	}{
		{0, 0x0000000000000000},
		{-0, 0x0000000000000000},
		{1, 0x0100000000010000},
		{-1, 0x0180000000010000},
		{65536, 0x0300000000010000},
		{-65536, 0x0380000000010000},
		{16777216, 0x0400000000010000},
		{-16777216, 0x0480000000010000},
	}

	for x, test := range tests {
		n := big.NewInt(test.in)
		r := BigToCompact(n)
		if r != test.out {
			t.Errorf("TestBigToCompact test #%d failed: got 0x%016x want 0x%016x\n",
				x, r, test.out)
			return
		}
	}

	// btm PowMin test
	// PowMinBits = 2161727821138738707, i.e 0x1e000000000dbe13, as defined
	// in /consensus/general.go
	n := big.NewInt(0).Lsh(big.NewInt(0x0dbe13), 27*8)
	out := uint64(0x1e000000000dbe13)
	r := BigToCompact(n)
	if r != out {
		t.Errorf("TestBigToCompact test #%d failed: got 0x%016x want 0x%016x\n",
			len(tests), r, out)
		return
	}
}
