import { render } from "@testing-library/svelte";
import { describe, it, expect } from "vitest";
import "@testing-library/jest-dom";
import ActivePlayerMessage from "../../src/components/ActivePlayerMessage.svelte";

describe("ActivePlayerMessage", () => {
  it("handles no active users", () => {
    const { getByText } = render(ActivePlayerMessage, { props: { count: 0 } });
    expect(getByText("The server has no active players.")).toBeInTheDocument();
  });

  it("handles 1 active user", () => {
    const { getByText } = render(ActivePlayerMessage, { props: { count: 1 } });
    expect(getByText("The server has 1 active player:")).toBeInTheDocument();
  });

  it("handles multiple active users", () => {
    const { getByText } = render(ActivePlayerMessage, { props: { count: 3 } });
    expect(getByText("The server has 3 active players:")).toBeInTheDocument();
  });
});
