import ActivePlayerMessage from "../../src/components/ActivePlayerMessage.svelte";
import { render } from '@testing-library/svelte';
import { expect } from 'chai';

describe(ActivePlayerMessage.name, () => {
  it("handles no active users", () => {
    const { getByText } = render(ActivePlayerMessage, { count: 0 });

    expect(getByText('The server has no active players.')).to.exist;
  });

  it("handles 1 active user", () => {
    const { getByText } = render(ActivePlayerMessage, { count: 1 });

    expect(getByText('The server has 1 active player:')).to.exist;
  });

  it("handles multiple active users", () => {
    const { getByText } = render(ActivePlayerMessage, { count: 3 });

    expect(getByText('The server has 3 active players:')).to.exist;
  });
});
