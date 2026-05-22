package calculators_test

import (
	"math"
	"routing-service/internal/calculators"
	"testing"
)

// Known reference pairs computed with standard Haversine formula.
var haversineTestCases = []struct {
	name    string
	lat1    float64
	lng1    float64
	lat2    float64
	lng2    float64
	wantKm  float64
	tolerKm float64
}{
	{
		name: "same point",
		lat1: -6.2, lng1: 106.8,
		lat2: -6.2, lng2: 106.8,
		wantKm:  0,
		tolerKm: 1e-9,
	},
	{
		// Jakarta → Bandung ≈ 120 km straight-line
		name: "Jakarta to Bandung",
		lat1: -6.2088, lng1: 106.8456,
		lat2: -6.9175, lng2: 107.6191,
		wantKm:  120.0,
		tolerKm: 5.0,
	},
	{
		// Equator crossing: (0,0) → (0,1) ≈ 111.195 km
		name: "equator 1 degree longitude",
		lat1: 0, lng1: 0,
		lat2: 0, lng2: 1,
		wantKm:  111.195,
		tolerKm: 0.1,
	},
	{
		// Antipodal points: half Earth circumference ≈ 20015 km
		name: "antipodal",
		lat1: 0, lng1: 0,
		lat2: 0, lng2: 180,
		wantKm:  20015.0,
		tolerKm: 5.0,
	},
}

func TestHaversineDistance(t *testing.T) {
	for _, tc := range haversineTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := calculators.HaversineDistance(tc.lat1, tc.lng1, tc.lat2, tc.lng2)
			if math.Abs(got-tc.wantKm) > tc.tolerKm {
				t.Errorf("HaversineDistance(%v,%v → %v,%v) = %.4f km, want %.4f ± %.4f",
					tc.lat1, tc.lng1, tc.lat2, tc.lng2, got, tc.wantKm, tc.tolerKm)
			}
		})
	}
}

func TestHaversineDistance_Symmetry(t *testing.T) {
	// d(A,B) == d(B,A)
	a2b := calculators.HaversineDistance(-6.2088, 106.8456, -6.9175, 107.6191)
	b2a := calculators.HaversineDistance(-6.9175, 107.6191, -6.2088, 106.8456)
	if math.Abs(a2b-b2a) > 1e-9 {
		t.Errorf("Haversine not symmetric: %.9f vs %.9f", a2b, b2a)
	}
}

func TestHaversineDistance_NonNegative(t *testing.T) {
	pairs := [][4]float64{
		{0, 0, 0, 0},
		{-90, -180, 90, 180},
		{45, 45, -45, -45},
	}
	for _, p := range pairs {
		d := calculators.HaversineDistance(p[0], p[1], p[2], p[3])
		if d < 0 {
			t.Errorf("negative distance %.9f for (%v,%v)→(%v,%v)", d, p[0], p[1], p[2], p[3])
		}
	}
}
