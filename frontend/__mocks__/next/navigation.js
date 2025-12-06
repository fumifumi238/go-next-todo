const mockPush = jest.fn();

module.exports = {
  useRouter: jest.fn(() => ({
    push: mockPush,
    replace: jest.fn(),
    refresh: jest.fn(),
    back: jest.fn(),
    forward: jest.fn(),
  })),
  usePathname: jest.fn(() => "/mock-path"),
  useSearchParams: jest.fn(() => new URLSearchParams()),
};
