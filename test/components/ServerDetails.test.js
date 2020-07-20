import ServerDetails from "../../src/components/ServerDetails.svelte";
import { render } from '@testing-library/svelte';
import { expect } from 'chai';

describe(ServerDetails.name, () => {
  describe("server details returned sucessfully", () => {
    it("handles resolved promise", async () => {
      const serverDetails = Promise.resolve({
        players: {
            max: 20,
            online: 0,
        }
      });
      const { getByText } = render(ServerDetails, {
          serverDetails
      });

      await serverDetails;

      expect(getByText('The server has no active players.')).to.exist;
    });

    it("handles no active users", () => {
      const { getByText } = render(ServerDetails, {
          serverDetails: {
            players: {
                max: 20,
                online: 0,
            }
          }
      });

      expect(getByText('The server has no active players.')).to.exist;
    });

    it("handles 1 active user", () => {
      const { getByText } = render(ServerDetails, {
          serverDetails: {
              players: {
                  max: 20,
                  online: 1,
                  sample: [{
                      id: "cdce37cd-2215-42ef-a4a4-c8b9189c9259",
                      name: "example"
                  }]
              }
          }
      });

      expect(getByText('The server has 1 active player:')).to.exist;
      expect(getByText('example')).to.exist;
    });

    it("handles multiple active users", () => {
      const { getByText } = render(ServerDetails, {
          serverDetails: {
              players: {
                  max: 20,
                  online: 2,
                  sample: [
                      {
                          id: "cdce37cd-2215-42ef-a4a4-c8b9189c9259",
                          name: "example"
                      },
                      {
                          id: "d720a93f-da90-41fa-8653-d09d81fa4b77",
                          name: "example2"
                      }
                  ]
              }
          }
      });

      expect(getByText('The server has 2 active players:')).to.exist;
      expect(getByText('example')).to.exist;
      expect(getByText('example2')).to.exist;
    });

  });

  describe("server details are still loading", () => {
    it("displays loading message", () => {
      const serverDetails = new Promise(() => {});
      const { getByText } = render(ServerDetails, {
          serverDetails
      });

      expect(getByText('Loading details...')).to.exist;
    });
  });

  describe("server details failed to load", () => {
    it("displays an error message", async () => {
      const serverDetails = Promise.reject("Error");
      const { getByText } = render(ServerDetails, {
          serverDetails
      });

      await serverDetails.catch(() => {});

      expect(getByText('Failed to load details.')).to.exist;
      expect(getByText('Failed to load details.').attributes[0].value).to.contain('error');
    });
  });
});
