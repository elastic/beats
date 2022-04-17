import { journey } from '@menderesk/synthetics';
import {
  loadAppStep,
  addTaskStep,
  assertTaskListSizeStep,
  checkForTaskStep,
  destroyTaskStep,
} from './helpers';

journey('basic addition and completion of single task', async ({ page }) => {
  const testText = "Don't put salt in your eyes";

  loadAppStep(page);
  addTaskStep(page, testText);
  assertTaskListSizeStep(page, 1);
  checkForTaskStep(page, testText);
  destroyTaskStep(page, testText);
  assertTaskListSizeStep(page, 0);
});

journey('adding and removing a few tasks', async ({ page }) => {
  const testTasks = ['Task 1', 'Task 2', 'Task 3'];

  loadAppStep(page);
  testTasks.forEach(t => {
    addTaskStep(page, t);
  });

  assertTaskListSizeStep(page, 3);

  // remove the middle task and check that it worked
  destroyTaskStep(page, testTasks[1]);
  assertTaskListSizeStep(page, 2);

  // add a new task and check it exists
  addTaskStep(page, 'Task 4');
  assertTaskListSizeStep(page, 3);
});
