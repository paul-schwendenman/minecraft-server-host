import { render } from "@testing-library/svelte";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { writable } from "svelte/store";
import "@testing-library/jest-dom";
import ServerStatus from "./ServerStatus.svelte";

const mockStatus = writable({
  instance: { state: "stopped", ip_address: null },
  dns_record: {},
});

vi.mock("@minecraft/data", async () => {
  const actual = await vi.importActual("@minecraft/data");
  return {
    ...actual,
    status: mockStatus,
  };
});

describe("ServerStatus", () => {
  beforeEach(() => {
    mockStatus.set({
      instance: { state: "stopped", ip_address: null },
      dns_record: {},
    });
  });

  describe("base", () => {
    it("displays DNS name as heading", () => {
      mockStatus.set({
        instance: { state: "stopped", ip_address: null },
        dns_record: { name: "example.test" },
      });

      const { getByRole } = render(ServerStatus);
      expect(
        getByRole("heading", { name: "example.test" }),
      ).toBeInTheDocument();
    });

    it("displays instance state", () => {
      mockStatus.set({
        instance: { state: "terminated" },
        dns_record: {},
      });

      const { getByText } = render(ServerStatus);
      expect(getByText("Server is terminated.")).toBeInTheDocument();
    });

    it("has refresh button", () => {
      mockStatus.set({
        instance: { state: "stopped" },
        dns_record: {},
      });

      const { getByRole } = render(ServerStatus);
      expect(getByRole("button", { name: "Refresh" })).toBeInTheDocument();
    });
  });

  describe("running", () => {
    it("has a stop button", () => {
      mockStatus.set({
        instance: { state: "running" },
        dns_record: {},
      });

      const { getByRole } = render(ServerStatus);
      expect(getByRole("button", { name: "Stop" })).toBeInTheDocument();
    });

    it("displays the current IP address", () => {
      mockStatus.set({
        instance: { state: "running", ip_address: "10.0.0.1" },
        dns_record: {},
      });

      const { getByText } = render(ServerStatus);
      expect(getByText(/IP address:/)).toHaveTextContent(
        "IP address: 10.0.0.1",
      );
    });
  });

  describe("stopped", () => {
    it("has a start button", () => {
      mockStatus.set({
        instance: { state: "stopped" },
        dns_record: {},
      });

      const { getByRole } = render(ServerStatus);
      expect(getByRole("button", { name: "Start" })).toBeInTheDocument();
    });
  });

  describe("mismatched DNS", () => {
    it("allows updating of DNS record", () => {
      mockStatus.set({
        instance: { state: "running", ip_address: "10.0.0.1" },
        dns_record: { value: "10.0.0.2" },
      });

      const { getByRole } = render(ServerStatus);
      expect(getByRole("button", { name: "Update DNS" })).toBeInTheDocument();
    });
  });
});

