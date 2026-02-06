// Authentication functionality
document.addEventListener('DOMContentLoaded', () => {
    const signupForm = document.querySelector('#signup-form');
    const loginForm = document.querySelector('#login-form');
    const passwordInput = document.querySelector('input[type="password"]');
    const githubOAuthBtn = document.querySelector('.github-oauth');
    const googleOAuthBtn = document.querySelector('.google-oauth');

    // Password strength indicator
    if (passwordInput && signupForm) {
        const strengthBar = document.querySelector('.password-strength-bar');
        
        passwordInput.addEventListener('input', (e) => {
            const password = e.target.value;
            const strength = calculatePasswordStrength(password);
            
            strengthBar.className = 'password-strength-bar';
            if (password.length > 0) {
                if (strength < 40) {
                    strengthBar.classList.add('weak');
                } else if (strength < 70) {
                    strengthBar.classList.add('medium');
                } else {
                    strengthBar.classList.add('strong');
                }
            }
        });
    }

    function calculatePasswordStrength(password) {
        let strength = 0;
        
        if (password.length >= 8) strength += 25;
        if (password.length >= 12) strength += 15;
        if (/[a-z]/.test(password)) strength += 15;
        if (/[A-Z]/.test(password)) strength += 15;
        if (/[0-9]/.test(password)) strength += 15;
        if (/[^a-zA-Z0-9]/.test(password)) strength += 15;
        
        return strength;
    }

    // Signup form submission
    if (signupForm) {
        signupForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const formData = new FormData(signupForm);
            const data = {
                name: formData.get('name'),
                email: formData.get('email'),
                password: formData.get('password'),
                company: formData.get('company'),
                newsletter: formData.get('newsletter') === 'on'
            };
            
            // Validation
            if (!validateEmail(data.email)) {
                showError('Please enter a valid email address');
                return;
            }
            
            if (data.password.length < 8) {
                showError('Password must be at least 8 characters long');
                return;
            }
            
            if (!formData.get('terms')) {
                showError('You must agree to the Terms of Service and Privacy Policy');
                return;
            }
            
            // Submit signup
            await submitSignup(data);
        });
    }

    // Login form submission
    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const formData = new FormData(loginForm);
            const data = {
                email: formData.get('email'),
                password: formData.get('password'),
                remember: formData.get('remember') === 'on'
            };
            
            // Validation
            if (!validateEmail(data.email)) {
                showError('Please enter a valid email address');
                return;
            }
            
            if (!data.password) {
                showError('Please enter your password');
                return;
            }
            
            // Submit login
            await submitLogin(data);
        });
    }

    // OAuth handlers
    if (githubOAuthBtn) {
        githubOAuthBtn.addEventListener('click', () => {
            initiateOAuth('github');
        });
    }

    if (googleOAuthBtn) {
        googleOAuthBtn.addEventListener('click', () => {
            initiateOAuth('google');
        });
    }

    // Helper functions
    function validateEmail(email) {
        const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return re.test(email);
    }

    function showError(message) {
        let errorEl = document.querySelector('.form-error');
        if (!errorEl) {
            errorEl = document.createElement('div');
            errorEl.className = 'form-error';
            const form = signupForm || loginForm;
            form.insertBefore(errorEl, form.firstChild);
        }
        errorEl.textContent = message;
        errorEl.style.display = 'block';
        
        // Scroll to error
        errorEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        
        setTimeout(() => {
            errorEl.style.display = 'none';
        }, 5000);
    }

    function showSuccess(message) {
        let successEl = document.querySelector('.form-success');
        if (!successEl) {
            successEl = document.createElement('div');
            successEl.className = 'form-success';
            const form = signupForm || loginForm;
            form.insertBefore(successEl, form.firstChild);
        }
        successEl.textContent = message;
        successEl.style.display = 'block';
        
        // Scroll to success message
        successEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }

    function setButtonLoading(button, loading) {
        if (loading) {
            button.classList.add('loading');
            button.disabled = true;
        } else {
            button.classList.remove('loading');
            button.disabled = false;
        }
    }

    async function submitSignup(data) {
        const submitBtn = signupForm.querySelector('button[type="submit"]');
        setButtonLoading(submitBtn, true);
        
        try {
            const response = await fetch('/api/auth/signup', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data),
            });
            
            const result = await response.json();
            
            if (response.ok) {
                // Track signup event
                if (window.gtag) {
                    window.gtag('event', 'sign_up', {
                        method: 'email'
                    });
                }
                
                // Show success and redirect
                showSuccess('Account created successfully! Redirecting...');
                
                setTimeout(() => {
                    // Redirect to email verification or dashboard
                    if (result.requiresVerification) {
                        window.location.href = '/verify-email?email=' + encodeURIComponent(data.email);
                    } else {
                        window.location.href = '/dashboard';
                    }
                }, 1500);
            } else {
                showError(result.message || 'Signup failed. Please try again.');
            }
        } catch (error) {
            console.error('Signup error:', error);
            showError('Network error. Please check your connection and try again.');
        } finally {
            setButtonLoading(submitBtn, false);
        }
    }

    async function submitLogin(data) {
        const submitBtn = loginForm.querySelector('button[type="submit"]');
        setButtonLoading(submitBtn, true);
        
        try {
            const response = await fetch('/api/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data),
            });
            
            const result = await response.json();
            
            if (response.ok) {
                // Store token if remember me is checked
                if (data.remember && result.token) {
                    localStorage.setItem('auth_token', result.token);
                } else if (result.token) {
                    sessionStorage.setItem('auth_token', result.token);
                }
                
                // Track login event
                if (window.gtag) {
                    window.gtag('event', 'login', {
                        method: 'email'
                    });
                }
                
                // Redirect to dashboard
                showSuccess('Login successful! Redirecting...');
                setTimeout(() => {
                    window.location.href = result.redirectUrl || '/dashboard';
                }, 1000);
            } else {
                showError(result.message || 'Login failed. Please check your credentials.');
            }
        } catch (error) {
            console.error('Login error:', error);
            showError('Network error. Please check your connection and try again.');
        } finally {
            setButtonLoading(submitBtn, false);
        }
    }

    function initiateOAuth(provider) {
        // Track OAuth attempt
        if (window.gtag) {
            window.gtag('event', 'oauth_initiated', {
                provider: provider
            });
        }
        
        // Redirect to OAuth endpoint
        const redirectUrl = encodeURIComponent(window.location.origin + '/auth/callback');
        window.location.href = `/api/auth/oauth/${provider}?redirect=${redirectUrl}`;
    }

    // Handle OAuth callback
    const urlParams = new URLSearchParams(window.location.search);
    const oauthError = urlParams.get('error');
    const oauthSuccess = urlParams.get('success');
    
    if (oauthError) {
        showError(decodeURIComponent(oauthError));
        // Clean URL
        window.history.replaceState({}, document.title, window.location.pathname);
    }
    
    if (oauthSuccess) {
        showSuccess('Authentication successful! Redirecting...');
        setTimeout(() => {
            window.location.href = '/dashboard';
        }, 1000);
    }

    // Email verification resend
    const resendBtn = document.querySelector('.resend-verification');
    if (resendBtn) {
        let resendCooldown = 0;
        
        resendBtn.addEventListener('click', async () => {
            if (resendCooldown > 0) return;
            
            const email = urlParams.get('email');
            if (!email) {
                showError('Email address not found');
                return;
            }
            
            try {
                const response = await fetch('/api/auth/resend-verification', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ email }),
                });
                
                if (response.ok) {
                    showSuccess('Verification email sent! Please check your inbox.');
                    
                    // Set cooldown
                    resendCooldown = 60;
                    resendBtn.disabled = true;
                    resendBtn.textContent = `Resend in ${resendCooldown}s`;
                    
                    const interval = setInterval(() => {
                        resendCooldown--;
                        if (resendCooldown <= 0) {
                            clearInterval(interval);
                            resendBtn.disabled = false;
                            resendBtn.textContent = 'Resend verification email';
                        } else {
                            resendBtn.textContent = `Resend in ${resendCooldown}s`;
                        }
                    }, 1000);
                } else {
                    const result = await response.json();
                    showError(result.message || 'Failed to resend verification email');
                }
            } catch (error) {
                console.error('Resend error:', error);
                showError('Network error. Please try again.');
            }
        });
    }

    // Password reset form
    const resetForm = document.querySelector('#reset-password-form');
    if (resetForm) {
        resetForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const email = resetForm.querySelector('input[type="email"]').value;
            
            if (!validateEmail(email)) {
                showError('Please enter a valid email address');
                return;
            }
            
            const submitBtn = resetForm.querySelector('button[type="submit"]');
            setButtonLoading(submitBtn, true);
            
            try {
                const response = await fetch('/api/auth/reset-password', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ email }),
                });
                
                if (response.ok) {
                    showSuccess('Password reset instructions sent to your email!');
                    resetForm.reset();
                } else {
                    const result = await response.json();
                    showError(result.message || 'Failed to send reset instructions');
                }
            } catch (error) {
                console.error('Reset error:', error);
                showError('Network error. Please try again.');
            } finally {
                setButtonLoading(submitBtn, false);
            }
        });
    }

    // Auto-focus first input
    const firstInput = document.querySelector('.auth-form input:not([type="checkbox"])');
    if (firstInput) {
        firstInput.focus();
    }
});
