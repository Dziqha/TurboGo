/** @type {import('next-sitemap').IConfig} */
module.exports = {
  siteUrl: "https://turbogo.web.id",
  generateRobotsTxt: true,
  generateIndexSitemap: false,
  exclude: ["/api/*", "/admin/*"],
  robotsTxtOptions: {
    policies: [
      {
        userAgent: "*",
        allow: "/",
      },
      {
        userAgent: "Googlebot",
        allow: "/",
      },
    ],
    additionalSitemaps: ["https://turbogo.web.id/sitemap.xml"],
  },
};
