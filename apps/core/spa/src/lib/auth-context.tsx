import { createContext, useContext, type ReactNode } from 'react'

export type AuthUser = {
  id: string
  orgId: string
  email: string
  name: string
  roles: string[]
}

export type AuthContextValue = {
  user: AuthUser
}

export const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ user, children }: { user: AuthUser; children?: ReactNode }) {
  return <AuthContext.Provider value={{ user }}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth() requires <AuthProvider>')
  return ctx
}

/** Signed-in user. Requires `<AuthProvider>`. */
export function getCurrentUser(): AuthUser {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('getCurrentUser() requires <AuthProvider>')
  return ctx.user
}
