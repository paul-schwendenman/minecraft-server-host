import fetchMock from 'fetch-mock';
import chai, { expect } from 'chai';
import chaiAsPromised from 'chai-as-promised';
import * as dataService from '../src/data.service';

chai.use(chaiAsPromised);

describe("data.service", () => {
    afterEach(() => {
        fetchMock.reset();
    });
    describe("getStatus", () => {
        it("fetches server status from backend", async () => {
            fetchMock
                .get('/__baseUrl__/status', { dns_record: { name: "minecraft.test"}});

            const result = await dataService.getStatus();

            expect(result).to.deep.equal({ dns_record: {name: "minecraft.test"}});
        });

        it("catches failure to fetch server status", () => {
            fetchMock
                .get('/__baseUrl__/status', 500);

            const result = dataService.getStatus();

            return expect(result).to.be.rejected;
        });
    });
});
