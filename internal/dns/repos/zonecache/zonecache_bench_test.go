package zonecache

import (
	"testing"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

func BenchmarkZoneCache_PutZone_SingleRecord(b *testing.B) {
	zc := New()
	records := []domain.ResourceRecord{
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		zc.PutZone("example.com.", records)
	}
}

func BenchmarkZoneCache_PutZone_MultipleRecords(b *testing.B) {
	zc := New()
	records := []domain.ResourceRecord{
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		{Name: "mail.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}},
		{Name: "ftp.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 3}},
		{Name: "api.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 4}},
		{Name: "example.com.", Type: 15, Class: 1, Data: []byte{10, 0, 'm', 'a', 'i', 'l'}},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		zc.PutZone("example.com.", records)
	}
}

func BenchmarkZoneCache_PutZone_LargeZone(b *testing.B) {
	zc := New()

	// Create a large zone with 100 records
	records := make([]domain.ResourceRecord, 100)
	for i := 0; i < 100; i++ {
		records[i] = domain.ResourceRecord{
			Name:  "host" + string(rune('0'+i%10)) + ".example.com.",
			Type:  1,
			Class: 1,
			Data:  []byte{192, 168, 1, byte(i)},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		zc.PutZone("example.com.", records)
	}
}

func BenchmarkZoneCache_FindRecords_Hit(b *testing.B) {
	zc := New()
	records := []domain.ResourceRecord{
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		{Name: "mail.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}},
		{Name: "ftp.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 3}},
	}
	zc.PutZone("example.com.", records)

	query := domain.Question{
		Name:  "www.example.com.",
		Type:  1,
		Class: 1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = zc.FindRecords(query)
	}
}

func BenchmarkZoneCache_FindRecords_Miss(b *testing.B) {
	zc := New()
	records := []domain.ResourceRecord{
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
	}
	zc.PutZone("example.com.", records)

	query := domain.Question{
		Name:  "nonexistent.example.com.",
		Type:  1,
		Class: 1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = zc.FindRecords(query)
	}
}

func BenchmarkZoneCache_FindRecords_WrongZone(b *testing.B) {
	zc := New()
	records := []domain.ResourceRecord{
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
	}
	zc.PutZone("example.com.", records)

	query := domain.Question{
		Name:  "www.different.com.",
		Type:  1,
		Class: 1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = zc.FindRecords(query)
	}
}

func BenchmarkZoneCache_FindRecords_MultipleRecords(b *testing.B) {
	zc := New()
	records := []domain.ResourceRecord{
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}},
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 3}},
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 4}},
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 5}},
	}
	zc.PutZone("example.com.", records)

	query := domain.Question{
		Name:  "www.example.com.",
		Type:  1,
		Class: 1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = zc.FindRecords(query)
	}
}

func BenchmarkZoneCache_RemoveZone(b *testing.B) {
	records := []domain.ResourceRecord{
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		{Name: "mail.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		zc := New()
		zc.PutZone("example.com.", records) // Setup for each iteration
		zc.RemoveZone("example.com.")
	}
}

func BenchmarkZoneCache_Zones(b *testing.B) {
	zc := New()
	records := []domain.ResourceRecord{
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
	}

	// Setup multiple zones
	zc.PutZone("example.com.", records)
	zc.PutZone("test.com.", records)
	zc.PutZone("demo.org.", records)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = zc.Zones()
	}
}

func BenchmarkZoneCache_Count(b *testing.B) {
	zc := New()
	records := []domain.ResourceRecord{
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		{Name: "mail.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}},
		{Name: "ftp.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 3}},
	}
	zc.PutZone("example.com.", records)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = zc.Count()
	}
}

func BenchmarkZoneCache_ConcurrentReads(b *testing.B) {
	zc := New()
	records := []domain.ResourceRecord{
		{Name: "www.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		{Name: "mail.example.com.", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}},
	}
	zc.PutZone("example.com.", records)

	query := domain.Question{
		Name:  "www.example.com.",
		Type:  1,
		Class: 1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = zc.FindRecords(query)
		}
	})
}
