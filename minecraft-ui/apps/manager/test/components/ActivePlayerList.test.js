import { render } from "@testing-library/svelte";
import { describe, it, expect } from "vitest";
import "@testing-library/jest-dom";
import ActivePlayerList from "../../src/components/ActivePlayerList.svelte";

describe("ActivePlayerList", () => {
  it("handles no active users", () => {
    const { container, queryAllByRole } = render(ActivePlayerList);

    expect(container).toHaveTextContent("");
    expect(queryAllByRole("listitem")).toHaveLength(0);
  });

  it("handles one active user", () => {
    const samplePlayers = [{ name: "Bob" }];
    const { getByText, getAllByRole } = render(ActivePlayerList, {
      props: { players: samplePlayers },
    });

    expect(getAllByRole("listitem")).toHaveLength(1);
    expect(getByText("Bob")).toBeInTheDocument();
  });

  it("handles multiple active users", () => {
    const samplePlayers = [{ name: "Bob" }, { name: "Bill" }];
    const { getByText, getAllByRole } = render(ActivePlayerList, {
      props: { players: samplePlayers },
    });

    expect(getAllByRole("listitem")).toHaveLength(2);
    expect(getByText("Bob")).toBeInTheDocument();
    expect(getByText("Bill")).toBeInTheDocument();
  });
});
