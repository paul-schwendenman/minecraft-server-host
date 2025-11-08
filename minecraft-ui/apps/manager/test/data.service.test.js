import { describe, it, expect, afterEach, vi } from "vitest";
import * as dataService from "../src/data.service";

describe("data.service", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("getStatus", () => {
    it("fetches server status from backend", async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve({ dns_record: { name: "minecraft.test" } }),
        }),
      );

      const result = await dataService.getStatus();
      expect(result).toEqual({ dns_record: { name: "minecraft.test" } });
    });

    it("catches failure to fetch server status", async () => {
      global.fetch = vi.fn(() => Promise.resolve({ ok: false, status: 500 }));

      await expect(dataService.getStatus()).rejects.toBeTruthy();
    });
  });
});
