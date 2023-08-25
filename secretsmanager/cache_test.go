package secretsmanager

import (
	"context"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	ctx := context.Background()

	ttl := 100 * time.Millisecond
	j := NewJanitor(ttl)
	j.Run(ctx, func() {})

	cur, _, _, found := j.getCache()
	if found {
		t.Fatal("expect 'found' be false", cur)
	}

	cur, prev, pen := secret{value: "cur"}, secret{value: "prev"}, secret{value: "pen"}

	j.setCache(cur, prev, pen)

	gcur, gprev, gpen, found := j.getCache()

	t.Log(gcur, gprev, gpen, found)
	if !found {
		t.Fatal("expect 'found' be true")
	}
	if cur != gcur {
		t.Fatalf("expect %v, %v be equals", cur, gcur)
	}
	if prev != gprev {
		t.Fatalf("expect %v, %v be equals", prev, gprev)
	}
	if pen != gpen {
		t.Fatalf("expect %v, %v be equals", pen, gpen)
	}

	time.Sleep(ttl + 100*time.Millisecond)

	cur, prev, pen, found = j.getCache()
	if found {
		t.Fatal("expect 'found' be false")
	}
	if cur.value != "" || prev.value != "" || pen.value != "" {
		t.Fatalf("expect cache values be empty: %v, %v, %v", cur, prev, pen)
	}
}
