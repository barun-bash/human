import type { Meta, StoryObj } from '@storybook/react';
import VerifyCodePage from '../../pages/VerifyCodePage';

const meta = {
  title: 'Pages/VerifyCode',
  component: VerifyCodePage,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof VerifyCodePage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
