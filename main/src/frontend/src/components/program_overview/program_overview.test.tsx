import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { I18nProvider } from "@/i18n/i18n";

import { ProgramOverview, type ProgramOverviewState } from "./program_overview";

const renderOverview = (state: ProgramOverviewState, archive = { count: 2, limit: 15 }) => {
  const onArchive = vi.fn().mockResolvedValue(undefined);
  const onRestore = vi.fn().mockResolvedValue(undefined);
  const onDelete = vi.fn().mockResolvedValue(undefined);
  render(
    <I18nProvider initialLocale="pt">
      <MemoryRouter>
        <ProgramOverview
          name="Força 8 semanas"
          coverUrl=""
          kicker="Pacote Premium"
          facts={["8 semanas"]}
          state={state}
          editTo="/trainer/programs/p1/edit"
          backTo="/trainer/programs"
          archive={{ ...archive, onArchive, onRestore }}
          onDelete={onDelete}
        />
      </MemoryRouter>
    </I18nProvider>
  );
  return { onArchive, onRestore, onDelete };
};

describe("ProgramOverview", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("shows the name, the state and an explicit Edit link to the builder", () => {
    renderOverview("published");
    expect(screen.getByRole("heading", { name: "Força 8 semanas" })).toBeTruthy();
    expect(screen.getByText("Publicado · à venda")).toBeTruthy();
    expect(screen.getByRole("link", { name: /Editar programa/ }).getAttribute("href")).toBe("/trainer/programs/p1/edit");
    expect(screen.getByText(/Arquivados: 2 de 15/)).toBeTruthy();
  });

  it("archives a published programme after confirmation", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const { onArchive } = renderOverview("published");
    fireEvent.click(screen.getByRole("button", { name: /Arquivar/ }));
    await waitFor(() => expect(onArchive).toHaveBeenCalledTimes(1));
  });

  it("blocks archiving at the limit of 15 and says why", () => {
    renderOverview("published", { count: 15, limit: 15 });
    expect((screen.getByRole("button", { name: /Arquivar/ }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByText(/Já tens 15 programas arquivados/)).toBeTruthy();
  });

  it("offers to put an archived programme back on the market", async () => {
    const { onRestore } = renderOverview("archived");
    expect(screen.queryByRole("button", { name: /^Arquivar/ })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /Voltar a pôr no mercado/ }));
    await waitFor(() => expect(onRestore).toHaveBeenCalledTimes(1));
  });

  it("does not delete without confirmation", () => {
    vi.spyOn(window, "confirm").mockReturnValue(false);
    const { onDelete } = renderOverview("draft");
    expect(screen.getByRole("link", { name: /Continuar a editar/ })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /Apagar/ }));
    expect(onDelete).not.toHaveBeenCalled();
  });
});
