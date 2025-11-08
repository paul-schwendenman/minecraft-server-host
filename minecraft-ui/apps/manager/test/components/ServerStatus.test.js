import { render } from "@testing-library/svelte";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { writable } from "svelte/store";
import "@testing-library/jest-dom";
import * as stores from "../../src/stores.js";
import ServerStatus from "../../src/components/ServerStatus.svelte";

describe("ServerStatus", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  describe("base", () => {
    it("displays DNS name as heading", () => {
      vi.spyOn(stores, "status", "get").mockReturnValue(
        writable({
          instance: { state: "stopped", ip_address: null },
          dns_record: { name: "example.test" },
        }),
      );

      const { getByRole } = render(ServerStatus);
      expect(
        getByRole("heading", { name: "example.test" }),
      ).toBeInTheDocument();
    });

    it("displays instance state", () => {
      vi.spyOn(stores, "status", "get").mockReturnValue(
        writable({
          instance: { state: "terminated" },
          dns_record: {},
        }),
      );

      const { getByText } = render(ServerStatus);
      expect(getByText("Server is terminated.")).toBeInTheDocument();
    });

    it("has refresh button", () => {
      vi.spyOn(stores, "status", "get").mockReturnValue(
        writable({
          instance: { state: "stopped" },
          dns_record: {},
        }),
      );

      const { getByRole } = render(ServerStatus);
      expect(getByRole("button", { name: "Refresh" })).toBeInTheDocument();
    });
  });

  describe("running", () => {
    it("has a stop button", () => {
      vi.spyOn(stores, "status", "get").mockReturnValue(
        writable({
          instance: { state: "running" },
          dns_record: {},
        }),
      );
      vi.spyOn(stores, "details", "get").mockReturnValue(
        writable({
          players: { online: 0, max: 20 },
        }),
      );

      const { getByRole } = render(ServerStatus);
      expect(getByRole("button", { name: "Stop" })).toBeInTheDocument();
    });

    it("displays the current IP address", () => {
      vi.spyOn(stores, "status", "get").mockReturnValue(
        writable({
          instance: { state: "running", ip_address: "10.0.0.1" },
          dns_record: {},
        }),
      );

      const { getByText } = render(ServerStatus);
      expect(getByText(/IP address:/)).toHaveTextContent(
        "IP address: 10.0.0.1",
      );
    });
  });

  describe("stopped", () => {
    it("has a start button", () => {
      vi.spyOn(stores, "status", "get").mockReturnValue(
        writable({
          instance: { state: "stopped" },
          dns_record: {},
        }),
      );

      const { getByRole } = render(ServerStatus);
      expect(getByRole("button", { name: "Start" })).toBeInTheDocument();
    });
  });

  describe("mismatched DNS", () => {
    it("allows updating of DNS record", () => {
      vi.spyOn(stores, "status", "get").mockReturnValue(
        writable({
          instance: { state: "running", ip_address: "10.0.0.1" },
          dns_record: { value: "10.0.0.2" },
        }),
      );

      const { getByRole } = render(ServerStatus);
      expect(getByRole("button", { name: "Update DNS" })).toBeInTheDocument();
    });
  });
});
