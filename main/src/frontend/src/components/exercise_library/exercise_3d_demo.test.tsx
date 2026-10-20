import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Exercise3DDemo, isSupportedThreeDKey } from "./exercise_3d_demo";

afterEach(cleanup);

beforeEach(() => {
  // jsdom has no WebGL; make the unsupported path explicit so the suite tests
  // the same fallback a browser without WebGL receives.
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockImplementation(
    (() => null) as typeof HTMLCanvasElement.prototype.getContext
  );
});

describe("isSupportedThreeDKey", () => {
  it("accepts the implemented scenes and rejects anything else", () => {
    expect(isSupportedThreeDKey("squat")).toBe(true);
    expect(isSupportedThreeDKey("push_up")).toBe(true);
    expect(isSupportedThreeDKey("biceps_curl")).toBe(true);
    expect(isSupportedThreeDKey("lunge")).toBe(true);
    expect(isSupportedThreeDKey("salsa")).toBe(false);
    expect(isSupportedThreeDKey(null)).toBe(false);
  });
});

describe("Exercise3DDemo", () => {
  it("renders nothing for exercises without a 3D demonstration", () => {
    const { container } = render(<Exercise3DDemo threeDKey={null} exerciseName="Bench Press" />);

    expect(container.firstChild).toBeNull();
  });

  it("renders nothing for an unknown stored key", () => {
    const { container } = render(<Exercise3DDemo threeDKey="unknown_scene" exerciseName="Bench Press" />);

    expect(container.firstChild).toBeNull();
  });

  it("loads the viewer only after the client opens it and falls back without WebGL", async () => {
    render(<Exercise3DDemo threeDKey="squat" exerciseName="Barbell Full Squat" />);

    expect(screen.queryByLabelText(/3D demonstration of/)).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: /Open 3D view/ }));

    expect(await screen.findByText(/3D previews are not supported on this device/)).toBeTruthy();
  });
});
