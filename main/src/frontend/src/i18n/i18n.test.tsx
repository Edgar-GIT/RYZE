import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { LanguagePrompt } from "@/components/language_prompt/language_prompt";
import { LanguageSwitcher } from "@/components/language_switcher/language_switcher";
import { formatMarketplacePrice } from "@/services/marketplace_api";

import { detectLocale, gettext, I18nProvider, useI18n } from "./i18n";

vi.mock("@utils/http_client", () => ({ apiPatch: vi.fn(() => Promise.resolve()) }));

const STORAGE_KEY = "ryze.locale";

const LangProbe = () => {
  const { locale } = useI18n();
  return <p data-testid="locale">{locale}</p>;
};

describe("detectLocale", () => {
  it("picks Portuguese for Portuguese time zones and pt-PT browsers", () => {
    expect(detectLocale("Europe/Lisbon", ["en-US"])).toBe("pt");
    expect(detectLocale("Atlantic/Azores", [])).toBe("pt");
    expect(detectLocale("America/New_York", ["pt-PT"])).toBe("pt");
  });

  it("picks English elsewhere, including Brazilian Portuguese", () => {
    expect(detectLocale("America/Sao_Paulo", ["pt-BR"])).toBe("en");
    expect(detectLocale("Europe/Madrid", ["es-ES"])).toBe("en");
  });
});

describe("gettext", () => {
  const catalog = { "Hello {name}": "Olá {name}" };

  it("translates with parameters in Portuguese", () => {
    expect(gettext("pt", [catalog], "Hello {name}", { name: "Ana" })).toBe("Olá Ana");
  });

  it("keeps English text, and falls back to it when an entry is missing", () => {
    expect(gettext("en", [catalog], "Hello {name}", { name: "Ana" })).toBe("Hello Ana");
    expect(gettext("pt", [catalog], "Not translated")).toBe("Not translated");
  });

  it("uses the first catalogue that has the entry", () => {
    expect(gettext("pt", [{ Save: "Guardar" }, { Save: "Gravar" }], "Save")).toBe("Guardar");
  });
});

describe("formatMarketplacePrice", () => {
  it("formats in the visitor's language", () => {
    expect(formatMarketplacePrice(1449, "EUR", "premium")).toBe("€14.49");
    expect(formatMarketplacePrice(1449, "EUR", "premium", "pt").replace(/\s/g, " ")).toBe("14,49 €");
    expect(formatMarketplacePrice(0, "EUR", "free", "pt")).toBe("Grátis");
  });
});

describe("language choice", () => {
  beforeEach(() => window.localStorage.clear());
  afterEach(() => {
    cleanup();
    window.localStorage.clear();
  });

  it("toggles the language with one click anywhere on the switch", () => {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ locale: "pt", source: "manual" }));
    render(
      <I18nProvider>
        <LanguageSwitcher variant="compact" />
        <LangProbe />
      </I18nProvider>
    );
    const toggle = screen.getByRole("button", { name: "Idioma: Português. Mudar para English" });
    expect(toggle.getAttribute("data-locale")).toBe("pt");

    fireEvent.click(toggle);
    expect(screen.getByTestId("locale").textContent).toBe("en");
    expect(JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? "{}")).toEqual({ locale: "en", source: "manual" });

    const english = screen.getByRole("button", { name: "Language: English. Switch to Português" });
    fireEvent.click(english);
    expect(screen.getByTestId("locale").textContent).toBe("pt");

    // Arrow keys pick a side without toggling past it.
    fireEvent.keyDown(english, { key: "ArrowRight" });
    expect(screen.getByTestId("locale").textContent).toBe("en");
    fireEvent.keyDown(english, { key: "ArrowRight" });
    expect(screen.getByTestId("locale").textContent).toBe("en");
    fireEvent.keyDown(english, { key: "ArrowLeft" });
    expect(screen.getByTestId("locale").textContent).toBe("pt");
  });

  it("asks once on the first visit and remembers the answer", () => {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ locale: "pt", source: "auto" }));
    const { unmount } = render(
      <I18nProvider>
        <LanguagePrompt />
        <LangProbe />
      </I18nProvider>
    );
    expect(screen.getByRole("dialog", { name: "RYZE em português" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Switch to English" }));
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(screen.getByTestId("locale").textContent).toBe("en");
    unmount();

    render(
      <I18nProvider>
        <LanguagePrompt />
      </I18nProvider>
    );
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("keeps the detected language when the visitor confirms it", () => {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ locale: "en", source: "auto" }));
    render(
      <I18nProvider>
        <LanguagePrompt />
        <LangProbe />
      </I18nProvider>
    );
    fireEvent.click(screen.getByRole("button", { name: "Continue in English" }));
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(screen.getByTestId("locale").textContent).toBe("en");
    expect(JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? "{}")).toEqual({ locale: "en", source: "manual" });
  });
});
