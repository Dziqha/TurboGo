package main

import (
	"time"

	"github.com/Dziqha/TurboGo"
	"github.com/Dziqha/TurboGo/core"
	"github.com/Dziqha/TurboGo/examples/controller"
	"github.com/Dziqha/TurboGo/middleware"
	"github.com/golang-jwt/jwt/v5"
)

func PublicHandler(c *core.Context) {
	// Simulasi logika bisnis ringan
	sum := 0
	for i := 0; i < 100_000; i++ {
		sum += i
	}
	buf := make([]byte, 1024*10) // alokasi memori biar gak diskip compiler
	copy(buf, []byte("TurboGo Cache Test"))

	c.JSON(200, map[string]any{
		"message": "Hello from PublicHandler!",
		"sum":     sum,
		"time":    time.Now().Format(time.RFC3339), // untuk bukti cache HIT tidak berubah
	})
}
func CobaPost(c *core.Context) {
	c.JSON(200, map[string]any{
		"message": "coba post",
	})
}
func PrivateHandler(c *core.Context) {
	c.JSON(200, map[string]any{"ok": true})
}

func GantiHandler(c *core.Context) {
	c.JSON(200, map[string]any{
		"data": "ganti",
	})
}

func HapusHandler(c *core.Context) {
	c.JSON(200, map[string]any{
		"data": "hapus",
	})
}

func HapusCacheHandler(c *core.Context) {
	sum := 0
	for i := 0; i < 1000; i++ {
		sum += i
	}
	data := sum // cegah compiler optimize

	c.JSON(200, map[string]any{
		"data": data,
	})
}

func HeavyHandler(c *core.Context) {
	start := time.Now()
	time.Sleep(2 * time.Second) // Simulasi kerja berat

	c.JSON(200, map[string]any{
		"message": "Heavy computation done",
		"at":      time.Now().Format(time.RFC3339),
		"took":    time.Since(start).String(),
	})
}

func AuthHandler(c *core.Context) {
	user := c.GetSession("user")
	if user == "" {
		c.Ctx.SetStatusCode(401)
		c.JSON(401, map[string]any{
			"error":   "unauthorized",
			"message": "Missing or invalid session",
		})
		return
	}

	c.JSON(200, map[string]any{
		"message": "Authenticated!",
		"user":    user,
	})
}

func LoginHandler(c *core.Context) {
	// Ambil username dari form (bisa pakai JSON parsing kalau mau)
	username := string(c.Ctx.FormValue("username"))
	if username == "" {
		c.JSON(400, map[string]any{"error": "username required"})
		return
	}

	// Buat claim bebas
	claims := jwt.MapClaims{
		"user_id": username,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	}

	// Buat token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte("supersecurekey123")) // ganti secret sesuai environment
	if err != nil {
		c.JSON(500, map[string]any{"error": "failed to sign token"})
		return
	}

	// Kirim ke client
	c.JSON(200, map[string]any{
		"token": signedToken,
	})
}

func main() {
	app := TurboGo.New()
	core.DisableLogger = true
	secret := "supersecurekey123"
	app.Use(
		middleware.Recover(),
	)

	// controller := controller.NewHandlerController()
	// data.NewRouter(
	// 	app,
	// 	controller,
	// )

	// // Init Queue & Pubsub engine (dengan Storage & Memory)
	// qe, _ := queue.NewEngine()
	// ps, _ := pubsub.NewEngine()

	// // Inject ke App
	// app.SetQueue(qe)
	// app.SetPubsub(ps)

	// // Route
	app.Post("/api/users", controller.CreateUserHandler).Named("create_user")

	// // Worker pakai storage
	// qe.RegisterWorkerAll("user:welcome-email", controller.SendWelcomeEmailWorker)
	// // Subscriber pakai storageps.Memory.Subscribe("user.created", controllers.OnUserCreated)
	// go func() {
	// 	ch := ps.Memory.Subscribe("user.created")
	// 	for msg := range ch {
	// 		// Panggil handler
	// 		if err := controller.OnUserCreated(msg); err != nil {
	// 			fmt.Println("❌ pubsub handler error:", err)
	// 		}
	// 	}
	// }()

	// go func() {
	// 	ch := ps.Storage.Subscribe("user.created")
	// 	for msg := range ch {
	// 		controller.OnUserCreated(msg)
	// 	}
	// }()
	ctx := app.InitEmptyEngine()
	app.WithPubsub(ctx)
	app.WithQueue(ctx)

	controller.Quehandler(ctx)
	go controller.PubsubHandler(ctx)

	auth := app.Group("/api", middleware.AuthJWT(secret))
	app.Post("/login", LoginHandler)
	auth.Get("/auth", AuthHandler).NoCache()
	app.Get("/public", PublicHandler)
	app.Post("/coba", CobaPost).Cache(3 * time.Second)
	app.Put("/ganti", GantiHandler)
	app.Delete("/hapus", HapusHandler)
	app.Delete("/hapus-cache", HapusCacheHandler).NoCache()

	// ⛔ No cache: override dengan .NoCache()
	app.Get("/private", PrivateHandler).NoCache()

	// ⚠️ POST normally no-cache, tapi bisa di-cache pakai .Cache()
	app.Post("/heavy", HeavyHandler).Cache(5 * time.Second)
	app.Handler()

	app.Get("/hello", func(c *core.Context) {

		c.Text(200, "Hello TurboGo!")
	}).NoCache()

	app.Get("/test-redis", func(c *core.Context) {
		c.Cache.Memory.Set("test", []byte("TurboGo"), 10*time.Second)

		val, _ := c.Cache.Memory.Get("test")
		if val == nil {
			c.JSON(404, map[string]string{"error": "not found"})
			return
		}

		c.JSON(200, map[string]string{
			"cache_value": string(val),
		})
	})

	app.Get("/ttl/:key", func(c *core.Context) {
		key := c.Param("key")
		ttl := c.Cache.Memory.TTL(key)

		switch ttl {
		case -2 * time.Second:
			c.JSON(404, map[string]string{"error": "key not found or expired"})
		case -1 * time.Second:
			c.JSON(200, map[string]string{"ttl": "infinite"})
		default:
			c.JSON(200, map[string]string{"ttl": ttl.String()})
		}
	})

	app.Get("/debug/routes", func(c *core.Context) {
		routes := app.RoutesInfo()
		c.JSON(200, routes)
	}).Named("routes")

	app.Get("/debug/redis/keys", func(c *core.Context) {
		keys := make([]string, 0)
		c.Cache.Memory.Range(func(k string, _ []byte) {
			keys = append(keys, k)
		})
		c.JSON(200, keys)
	})

	app.RunServer(":8080")
}
