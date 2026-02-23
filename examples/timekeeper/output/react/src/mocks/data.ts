// Auto-generated mock data factories based on Human IR Data Models

export const mockUser = (overrides?: Partial<any>) => ({
  id: 'mock-id-123',
  name: 'Jane Doe',
  email: 'jane.doe@example.com',
  password: 'Lorem ipsum dolor sit amet',
  reset_code: 'Lorem ipsum dolor sit amet',
  reset_code_expires: '2025-01-01T12:00:00Z',
  created: '2025-01-01T12:00:00Z',
  ...overrides,
});

export const mockUserList = (count: number = 3) => 
  Array.from({ length: count }).map((_, i) => mockUser({ id: `mock-id-${i}` }));

