import type { Meta, StoryObj } from '@storybook/react';
import ResetPasswordPage from '../../pages/ResetPasswordPage';

const meta = {
  title: 'Pages/ResetPassword',
  component: ResetPasswordPage,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof ResetPasswordPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
