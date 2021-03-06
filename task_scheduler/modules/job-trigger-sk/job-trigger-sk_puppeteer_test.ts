import * as path from 'path';
import { expect } from 'chai';
import {
  setUpPuppeteerAndDemoPageServer,
  takeScreenshot,
} from '../../../puppeteer-tests/util';

describe('job-trigger-sk', () => {
  const testBed = setUpPuppeteerAndDemoPageServer(
    path.join(__dirname, '..', '..', 'webpack.config.ts')
  );

  beforeEach(async () => {
    await testBed.page.goto(`${testBed.baseUrl}/dist/job-trigger-sk.html`);
    await testBed.page.setViewport({ width: 500, height: 600 });
  });

  it('should render the demo page (smoke test)', async () => {
    expect(await testBed.page.$$('job-trigger-sk')).to.have.length(1);
  });

  describe('screenshots', () => {
    it('shows the default view', async () => {
      await takeScreenshot(testBed.page, 'task_scheduler', 'job-trigger-sk');
    });
    it('deletes job from list', async () => {
      await testBed.page.click("delete-icon-sk");
      await takeScreenshot(testBed.page, 'task_scheduler', 'job-trigger-sk_deleted');
    });
    it('adds job to list', async () => {
      await testBed.page.click("add-icon-sk");
      await takeScreenshot(testBed.page, 'task_scheduler', 'job-trigger-sk_added');
    });
    it('triggers jobs', async () => {
      await testBed.page.type(".job_specs_input", "my-job");
      await testBed.page.type(".commit_input", "abc123");
      await takeScreenshot(testBed.page, 'task_scheduler', 'job-trigger-sk_pre-trigger');
      await testBed.page.click("send-icon-sk");
      await takeScreenshot(testBed.page, 'task_scheduler', 'job-trigger-sk_post-trigger');
    });
  });
});
