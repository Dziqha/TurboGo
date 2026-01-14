import Image from "next/image";
import Link from "next/link";

export default function HomePage() {
  return (
    <main className="min-h-screen px-6">
      <div className="mx-auto max-w-6xl pt-24 pb-24">
        {/* HERO */}
        <section className="grid grid-cols-1 gap-12 mb-24">
          {/* Brand */}
          <div className="mx-auto max-w-2xl text-center">
            <div className="flex justify-center mb-8">
              <Image
                src="/images/icon.png"
                alt="TurboGo"
                width={140}
                height={140}
                priority
                className="max-w-[128px]"
              />
            </div>

            <h1 className="text-5xl md:text-6xl font-medium text-neutral-900 dark:text-neutral-100 mb-6">
              TurboGo
            </h1>

            <p className="text-base text-neutral-600 dark:text-neutral-400 mb-10">
              High-performance Go framework powered by Tiered Zero-Copy Routing
            </p>

            <div className="flex justify-center gap-3">
              <Link
                href="/docs"
                className="px-6 py-3 rounded-md bg-black text-white dark:bg-white dark:text-black text-sm font-medium"
              >
                Get started
              </Link>

              <Link
                href="/docs"
                className="px-6 py-3 rounded-md border border-neutral-300 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300 text-sm font-medium"
              >
                Documentation
              </Link>
            </div>
          </div>
        </section>

        {/* DESCRIPTION */}
        <section className="mx-auto max-w-3xl mb-24">
          <p className="text-lg text-neutral-700 dark:text-neutral-300 leading-relaxed text-center">
            TurboGo is a modern backend framework designed for engineers who
            value performance, simplicity, and full control. Built on{" "}
            <span className="font-medium text-neutral-900 dark:text-neutral-100">
              Tiered Zero-Copy Routing (TZCR)
            </span>
            , it enables predictable request handling with minimal overhead.
          </p>
        </section>

        {/* FEATURES */}
        <section className="border-t border-neutral-200 dark:border-neutral-800 pt-20">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-12">
            {[
              {
                title: "Zero-Copy Routing",
                desc: "Tiered routing engine that resolves handlers without allocations.",
              },
              {
                title: "Built on fasthttp",
                desc: "Optimized for low latency and high concurrency workloads.",
              },
              {
                title: "Minimal Core",
                desc: "Clear APIs with no hidden magic or unnecessary abstraction.",
              },
              {
                title: "Production Ready",
                desc: "Queue, pubsub, caching, and graceful shutdown included.",
              },
            ].map((item, i) => (
              <div key={i}>
                <h3 className="text-sm font-medium text-neutral-900 dark:text-neutral-100 mb-3">
                  {item.title}
                </h3>
                <p className="text-sm text-neutral-600 dark:text-neutral-400 leading-relaxed">
                  {item.desc}
                </p>
              </div>
            ))}
          </div>
        </section>
      </div>
    </main>
  );
}
