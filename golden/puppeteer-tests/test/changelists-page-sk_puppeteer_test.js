const expect = require('chai').expect;
const addEventListenersToPuppeteerPage = require('./util').addEventListenersToPuppeteerPage;
const setUpPuppeteerAndDemoPageServer = require('./util').setUpPuppeteerAndDemoPageServer;
const takeScreenshot = require('./util').takeScreenshot;

describe('changelists-page-sk', () => {
  const testBed = setUpPuppeteerAndDemoPageServer(); // Contains page and baseUrl.

  beforeEach(async () => {
    const eventPromise = await addEventListenersToPuppeteerPage(testBed.page, ['end-task']);
    const loaded = eventPromise('end-task'); // Emitted when page is loaded.
    await testBed.page.goto(`${testBed.baseUrl}/dist/changelists-page-sk.html`);
    await loaded;
  });

  it('should render the demo page', async () => {
    // Smoke test.
    expect(await testBed.page.$$('changelists-page-sk')).to.have.length(1);
  });

  it('defaults to only open changelists', async () => {
    await testBed.page.setViewport({ width: 1200, height: 500 });
    await takeScreenshot(testBed.page, 'changelists-page-sk');
  });

  it('can show all changelists with a click', async () => {
    await testBed.page.setViewport({ width: 1200, height: 600 });
    await testBed.page.click('.controls checkbox-sk');
    await takeScreenshot(testBed.page, 'changelists-page-sk_show-all');
  });
});
