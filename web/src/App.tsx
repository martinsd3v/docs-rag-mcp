import { Routes, Route } from 'react-router-dom'
import { Layout } from './components/layout/Layout'
import { RequireProject } from './components/auth/RequireProject'
import { Dashboard } from './pages/Dashboard'
import { Documents } from './pages/Documents'
import { DocumentDetail } from './pages/DocumentDetail'
import { Search } from './pages/Search'
import { Upload } from './pages/Upload'
import { ProjectSelector } from './pages/ProjectSelector'

function App() {
  return (
    <Routes>
      {/* Project selection - no layout */}
      <Route path="/projects" element={<ProjectSelector />} />

      {/* Protected routes - require active project */}
      <Route element={<RequireProject />}>
        <Route element={<Layout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/documents" element={<Documents />} />
          <Route path="/documents/:id" element={<DocumentDetail />} />
          <Route path="/search" element={<Search />} />
          <Route path="/upload" element={<Upload />} />
        </Route>
      </Route>
    </Routes>
  )
}

export default App
