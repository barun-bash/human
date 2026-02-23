import type { Meta, StoryObj } from '@storybook/react';
import ForgotPasswordPage from '../../pages/ForgotPasswordPage';

const meta = {
  title: 'Pages/ForgotPassword',
  component: ForgotPasswordPage,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof ForgotPasswordPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
