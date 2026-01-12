import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App'
import posthog from 'posthog-js'

posthog.init('phc_8svND4SMHnm5j6VsW9kDdXLlqL3izJQ88rhssgy6CCb', {
    api_host: 'https://us.i.posthog.com',
    person_profiles: 'identified_only',
})

const rootElement = document.getElementById('root')
if (!rootElement) throw new Error('Failed to find the root element')

createRoot(rootElement).render(
    <StrictMode>
        <App />
    </StrictMode>,
)
