import type { Meta, StoryObj } from '@storybook/react';
import SignUpPage from '../../pages/SignUpPage';

const meta = {
  title: 'Pages/SignUp',
  component: SignUpPage,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof SignUpPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
