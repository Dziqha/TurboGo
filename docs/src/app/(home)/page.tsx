import Link from "next/link";
import Image from "next/image";

export default function HomePage() {
  return (
    <main className="flex flex-1 flex-col justify-center items-center min-h-screen relative overflow-hidden">
      <div className="relative z-10 mt-5 text-center px-4 max-w-6xl mx-auto">
        <div className="flex justify-center mb-6">
          <div className="relative">
            <Image
              src="/images/icon.png"
              alt="TurboGo TZCR Architecture"
              width={320}
              height={320}
              priority
              className="drop-shadow-2xl"
            />
            <div className="absolute inset-0 -z-10 rounded-full bg-blue-500/30 blur-3xl" />
          </div>
        </div>

        <h1 className="text-6xl md:text-7xl lg:text-8xl font-black mb-2 text-blue-600 leading-tight">
          TurboGo
        </h1>

        <p className="text-sm uppercase tracking-widest text-blue-500 font-semibold mb-6">
          Powered by Tiered Zero-Copy Routing (TZCR)
        </p>

        <p className="text-xl md:text-2xl text-gray-600 mb-10 max-w-3xl mx-auto leading-relaxed">
          A modern, developer-friendly backend framework built on{" "}
          <span className="text-blue-600 font-semibold">
            Tiered Zero-Copy Routing (TZCR)
          </span>{" "}
          — engineered for extreme speed, scalability, and minimalism to help
          you build powerful APIs with absolute efficiency.
        </p>

        <div className="flex flex-col sm:flex-row gap-4 justify-center items-center mb-14">
          <Link
            href="/docs"
            className="group relative px-8 py-4 bg-blue-600 hover:bg-blue-700 rounded-full font-semibold text-white shadow-lg shadow-blue-500/25 hover:shadow-blue-500/40 transition-all duration-300 transform hover:scale-105 hover:-translate-y-1"
          >
            <span className="relative z-10">Get Started</span>
          </Link>

          <Link
            href="/docs"
            className="group px-8 py-4 border border-blue-300 rounded-full font-semibold text-blue-600 hover:text-blue-700 hover:border-blue-400 transition-all duration-300 backdrop-blur-sm hover:bg-blue-50"
          >
            View Documentation
            <span className="ml-2 group-hover:translate-x-1 transition-transform duration-300 inline-block">
              →
            </span>
          </Link>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
          {[
            {
              title: "Tiered Zero-Copy Routing",
              desc: "Multi-layer routing engine with zero-allocation handler resolution for blazing-fast request dispatch.",
              icon: (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M3 12h6m6 0h6M9 6l6 6-6 6"
                />
              ),
            },
            {
              title: "Extreme Performance",
              desc: "Powered by fasthttp with custom routing optimized for ultra-low latency.",
              icon: (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13 10V3L4 14h7v7l9-11h-7z"
                />
              ),
            },
            {
              title: "Minimal, Yet Powerful",
              desc: "Clean API, clear structure — no clutter, full control over application flow.",
              icon: (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              ),
            },
            {
              title: "Ready for Scale",
              desc: "Built-in support for queueing, pubsub, caching, and graceful shutdowns.",
              icon: (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"
                />
              ),
            },
          ].map((item, idx) => (
            <div
              key={idx}
              className="group p-6 rounded-2xl bg-white/30 dark:bg-white/10 border border-neutral-200 dark:border-white/30 backdrop-blur-md shadow-md hover:border-neutral-300 dark:hover:border-white/50 transition-all duration-300 hover:scale-[1.03]"
            >
              <div className="w-12 h-12 bg-blue-600 rounded-xl flex items-center justify-center mb-4 group-hover:scale-110 transition-transform duration-300">
                <svg
                  className="w-6 h-6 text-white"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  {item.icon}
                </svg>
              </div>

              <h3 className="text-xl font-semibold text-neutral-800 dark:text-white mb-2">
                {item.title}
              </h3>
              <p className="text-neutral-700 dark:text-white/80">{item.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </main>
  );
}
