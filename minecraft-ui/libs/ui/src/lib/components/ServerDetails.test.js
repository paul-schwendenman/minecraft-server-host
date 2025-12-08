import { render } from "@testing-library/svelte";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { writable } from "svelte/store";
import "@testing-library/jest-dom";
import ServerDetails from "./ServerDetails.svelte";

const mockDetails = writable(null);

vi.mock("@minecraft/data", async () => {
  const actual = await vi.importActual("@minecraft/data");
  return {
    ...actual,
    details: mockDetails,
  };
});

describe("ServerDetails", () => {
  beforeEach(() => {
    mockDetails.set(null);
  });

  describe("server details returned successfully", () => {
    it("handles no active users", async () => {
      mockDetails.set({
        players: { max: 20, online: 0 },
        version: { name: "1.15.2" },
      });

      const { getByText } = render(ServerDetails);
      await new Promise((resolve) => setTimeout(resolve, 100));
      expect(
        getByText("The server has no active players."),
      ).toBeInTheDocument();
    });

    it("handles 1 active user", async () => {
      mockDetails.set({
        players: {
          max: 20,
          online: 1,
          sample: [{ id: "cdce37cd", name: "example" }],
        },
        version: { name: "1.15.2" },
      });

      const { getByText, getAllByRole } = render(ServerDetails);
      await new Promise((resolve) => setTimeout(resolve, 100));
      expect(getByText("The server has 1 active player:")).toBeInTheDocument();
      expect(getAllByRole("listitem")).toHaveLength(1);
      expect(getByText("example")).toBeInTheDocument();
    });

    it("handles multiple active users", async () => {
      mockDetails.set({
        players: {
          max: 20,
          online: 2,
          sample: [
            { id: "1", name: "example" },
            { id: "2", name: "example2" },
          ],
        },
        version: { name: "1.15.2" },
      });

      const { getByText, getAllByRole } = render(ServerDetails);
      await new Promise((resolve) => setTimeout(resolve, 100));
      expect(getByText("The server has 2 active players:")).toBeInTheDocument();
      expect(getAllByRole("listitem")).toHaveLength(2);
      expect(getByText("example")).toBeInTheDocument();
      expect(getByText("example2")).toBeInTheDocument();
    });
  });

  describe("server details are still loading", () => {
    it("displays loading message", () => {
      const pending = new Promise(() => {});
      mockDetails.set(pending);

      const { getByText } = render(ServerDetails);
      expect(getByText("Loading details...")).toBeInTheDocument();
    });
  });

  describe("server details failed to load", () => {
    it("displays an error message", async () => {
      mockDetails.set(Promise.reject("Error"));

      const { findByText } = render(ServerDetails);
      const errorEl = await findByText("Failed to load details.");
      expect(errorEl).toBeInTheDocument();
      expect(errorEl.className).toContain("text-red-700");
    });
  });
});

