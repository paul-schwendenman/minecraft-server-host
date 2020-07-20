import ServerStatus from "../../src/components/ServerStatus.svelte";
import { render } from '@testing-library/svelte';
import chai, { expect } from 'chai';
import chaiDom from 'chai-dom';

chai.use(chaiDom);

describe(ServerStatus.name, () => {
    describe("base", () => {
        it("displays dns name as heading", () => {
            const { getByRole } = render(ServerStatus, {
                serverStatus: {
                    dns_record: {
                        name: "example.test"
                    },
                    instance: {}
                },
                handleStart: () => {},
                handleStop: () => {},
                handleRefresh: () => {},
                handleSyncDNS: () => {},
            });

           expect(getByRole("heading", {name: "example.test"})).to.exist;
        });

        it("displays instance state", () => {
            const { getByText } = render(ServerStatus, {
                serverStatus: {
                    dns_record: {},
                    instance: {
                        state: "terminated"
                    }
                },
                handleStart: () => {},
                handleStop: () => {},
                handleRefresh: () => {},
                handleSyncDNS: () => {},
            });

           expect(getByText(/^Server is \w+.$/)).to.have.text("Server is terminated.");
        });

        it("has refresh button", () => {
            const { getByRole } = render(ServerStatus, {
                serverStatus: {
                    dns_record: {},
                    instance: {}
                },
                handleStart: () => {},
                handleStop: () => {},
                handleRefresh: () => {},
                handleSyncDNS: () => {},
            });

           expect(getByRole("button", {name: "Refresh"})).to.exist;
        });
    });

    describe("running", () => {
        it("has a stop button", () => {
            const { getByRole } = render(ServerStatus, {
                serverStatus: {
                    dns_record: {},
                    instance: {
                        state: "running"
                    }
                },
                handleStart: () => {},
                handleStop: () => {},
                handleRefresh: () => {},
                handleSyncDNS: () => {},
            });

           expect(getByRole("button", {name: "Stop"})).to.exist;
        });

        it("displays the current IP address", () => {
            const { getByText } = render(ServerStatus, {
                serverStatus: {
                    dns_record: {},
                    instance: {
                        state: "running",
                        ip_address: "10.0.0.1"
                    }
                },
                handleStart: () => {},
                handleStop: () => {},
                handleRefresh: () => {},
                handleSyncDNS: () => {},
            });

           expect(getByText(/IP address:/)).to.have.text('IP address: 10.0.0.1');

        });
    });

    describe("stopped", () => {
        it("has a start button", () => {
            const { getByRole } = render(ServerStatus, {
                serverStatus: {
                    dns_record: {},
                    instance: {
                        state: "stopped"
                    }
                },
                handleStart: () => {},
                handleStop: () => {},
                handleRefresh: () => {},
                handleSyncDNS: () => {},
            });

           expect(getByRole("button", {name: "Start"})).to.exist;
        });
    });

    describe("mismatched DNS", () => {
        it("allows updating of DNS record", () => {
            const { getByRole } = render(ServerStatus, {
                serverStatus: {
                    dns_record: {
                        value: "10.0.0.2"
                    },
                    instance: {
                        state: "running",
                        ip_address: "10.0.0.1"
                    }
                },
                handleStart: () => {},
                handleStop: () => {},
                handleRefresh: () => {},
                handleSyncDNS: () => {},
            });

           expect(getByRole("button", {name: "Update DNS Record"})).to.exist;
        });
    });
});
