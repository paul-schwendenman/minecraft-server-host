import ActivePlayerList from "../../src/components/ActivePlayerList.svelte";
import { render } from '@testing-library/svelte';
import chai, { expect } from 'chai';
import chaiDom from 'chai-dom';

chai.use(chaiDom);

describe(ActivePlayerList.name, () => {
  it("handles no active users", () => {
    const { container, queryAllByRole } = render(ActivePlayerList, { });

    expect(container).to.have.text('');
    expect(queryAllByRole('listitem')).to.be.empty;
  });

  it("handles one active user", () => {
    const samplePlayers = [
        { name: 'Bob'}
    ];
    const { getByText, getAllByRole } = render(ActivePlayerList, { players: samplePlayers});

    expect(getAllByRole('listitem')).to.have.lengthOf(1);
    expect(getByText('Bob')).to.exist;
  });

  it("handles multiple active users", () => {
    const samplePlayers = [
        { name: 'Bob'},
        { name: 'Bill'}
    ];
    const { getByText, getAllByRole } = render(ActivePlayerList, { players: samplePlayers});

    expect(getAllByRole('listitem')).to.have.lengthOf(2);
    expect(getByText('Bob')).to.exist;
    expect(getByText('Bill')).to.exist;
  });
});
