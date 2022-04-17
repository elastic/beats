import { journey, step } from '@menderesk/synthetics';
import { deepStrictEqual } from 'assert';
import { join } from 'path';

journey('check that title is present', async ({ page }) => {
  step('go to app', async () => {
    const path = 'file://' + join(__dirname, 'app', 'index.html');
    await page.goto(path);
  });

  step('check title is present', async () => {
    const header = await page.$('h1');
    deepStrictEqual(await header.textContent(), 'todos');
  });
});

journey('check that input placeholder is correct', async ({ page }) => {
  step('go to app', async () => {
    const path = 'file://' + join(__dirname, 'app', 'index.html');
    await page.goto(path);
  });

  step('check title is present', async () => {
    const input = await page.$('input.new-todo');
    deepStrictEqual(
      await input.getAttribute('placeholder'),
      'What nneeds to be done?'
    );
  });
});
