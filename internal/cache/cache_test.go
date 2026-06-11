package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInMem_SetAndGet(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	c.Set("key1", []byte("value1"), 0)
	val, ok := c.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", string(val))
}

func TestInMem_Get_Missing(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	_, ok := c.Get("nonexistent")
	assert.False(t, ok)
}

func TestInMem_Set_Overwrites(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	c.Set("key", []byte("old"), 0)
	c.Set("key", []byte("new"), 0)

	val, ok := c.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "new", string(val))
}

func TestInMem_Delete(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	c.Set("key", []byte("val"), 0)
	assert.True(t, c.Delete("key"))
	assert.False(t, c.Delete("nonexistent"))

	_, ok := c.Get("key")
	assert.False(t, ok)
}

func TestInMem_Exists(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	assert.False(t, c.Exists("key"))
	c.Set("key", []byte("val"), 0)
	assert.True(t, c.Exists("key"))
}

func TestInMem_TTL_NoExpiry(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	c.Set("perm", []byte("forever"), 0)
	assert.Equal(t, time.Duration(-1)*time.Second, c.TTL("perm"))
}

func TestInMem_TTL_NotFound(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	assert.Equal(t, time.Duration(-2)*time.Second, c.TTL("ghost"))
}

func TestInMem_TTL_Active(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	c.Set("temp", []byte("ephemeral"), time.Hour)
	ttl := c.TTL("temp")
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, time.Hour)
}

func TestInMem_TTL_Expired(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	c.Set("gone", []byte("bye"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	assert.Equal(t, time.Duration(-2)*time.Second, c.TTL("gone"))
}

func TestInMem_Expiry_GetReturnsFalse(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	c.Set("exp", []byte("will expire"), 50*time.Millisecond)
	time.Sleep(60 * time.Millisecond)

	_, ok := c.Get("exp")
	assert.False(t, ok)
}

func TestInMem_SetEx(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	c.SetEx("key", []byte("val"), 3600)
	val, ok := c.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "val", string(val))
}

func TestInMem_SetNX_NewKey(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	ok := c.SetNX("new", []byte("value"), 0)
	assert.True(t, ok)

	val, _ := c.Get("new")
	assert.Equal(t, "value", string(val))
}

func TestInMem_SetNX_ExistingKey(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	c.Set("existing", []byte("old"), 0)
	ok := c.SetNX("existing", []byte("new"), 0)
	assert.False(t, ok)

	val, _ := c.Get("existing")
	assert.Equal(t, "old", string(val))
}

func TestInMem_SetNX_ExpiredKey(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	c.Set("expired", []byte("old"), 50*time.Millisecond)
	time.Sleep(60 * time.Millisecond)

	ok := c.SetNX("expired", []byte("new"), 0)
	assert.True(t, ok)

	val, _ := c.Get("expired")
	assert.Equal(t, "new", string(val))
}

func TestInMem_Size(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	assert.Equal(t, 0, c.Size())
	c.Set("a", []byte("1"), 0)
	c.Set("b", []byte("2"), 0)
	assert.Equal(t, 2, c.Size())
}

func TestInMem_Range(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	c.Set("k1", []byte("v1"), 0)
	c.Set("k2", []byte("v2"), 0)

	collected := make(map[string]string)
	c.Range(func(key string, value []byte) {
		collected[key] = string(value)
	})

	assert.Equal(t, "v1", collected["k1"])
	assert.Equal(t, "v2", collected["k2"])
}

func TestInMem_Close_StopsCleaner(t *testing.T) {
	c := NewInMem()
	c.Set("key", []byte("val"), time.Hour)
	c.Close()

	// Cleaner goroutine should have exited
	time.Sleep(10 * time.Millisecond)
}

func TestInMem_Close_ClearsStore(t *testing.T) {
	c := NewInMem()
	c.Set("key", []byte("val"), 0)
	c.Close()

	assert.Equal(t, 0, c.Size())
}

func TestEngine_NewEngine(t *testing.T) {
	e, err := NewEngine()
	assert.NoError(t, err)
	assert.NotNil(t, e)
	assert.NotNil(t, e.Memory)
	e.Memory.Close()
}

func TestInMem_ConcurrentAccess(t *testing.T) {
	c := NewInMem()
	defer c.Close()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := string(rune('a' + n))
			c.Set(key, []byte("val"), 0)
			c.Get(key)
			c.Exists(key)
			c.Delete(key)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
