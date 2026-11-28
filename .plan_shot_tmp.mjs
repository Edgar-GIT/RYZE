import { chromium } from "playwright";
const [out, url, suffix = ""] = process.argv.slice(2);
const browser = await chromium.launch({ executablePath: "/usr/bin/chromium", args: ["--use-angle=swiftshader", "--enable-unsafe-swiftshader"] });
for (const [w, h, name] of [[1440, 900, "plan_desktop"], [390, 844, "plan_phone"]]) {
  const context = await browser.newContext({ viewport: { width: w, height: h } });
  await context.addInitScript(() => { try { localStorage.setItem("ryze.locale", JSON.stringify({ locale: "pt", source: "manual" })); } catch {} });
  const page = await context.newPage();
  await page.goto(url, { waitUntil: "networkidle" });
  const reject = page.getByRole("button", { name: /Rejeitar opcionais/ });
  await reject.first().waitFor({ timeout: 5000 }).catch(() => {});
  await reject.first().click().catch(() => {});
  await page.waitForTimeout(1000);
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  await page.screenshot({ path: `${out}/${name}${suffix}.png`, fullPage: name === "plan_phone" ? false : true });
  console.log(name, { overflow });
  await context.close();
}
await browser.close();
