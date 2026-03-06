import FormRecipient from "./generic/FormRecipient.js";

export default {
    name: 'AccountBusinessProfile',
    components: {
        FormRecipient
    },
    data() {
        return {
            type: window.TYPEUSER,
            phone: '',
            //
            jid: null,
            email: null,
            address: null,
            categories: [],
            profileOptions: {},
            businessHoursTimeZone: null,
            businessHours: [],
            //
            loading: false,
            dayNamesMap: {
                "MONDAY": "Segunda-feira",
                "TUESDAY": "Terça-feira",
                "WEDNESDAY": "Quarta-feira",
                "THURSDAY": "Quinta-feira",
                "FRIDAY": "Sexta-feira",
                "SATURDAY": "Sábado",
                "SUNDAY": "Domingo",
            }
        }
    },

    computed: {
        phone_id() {
            return this.phone + this.type;
        }
    },
    methods: {
        async openModal() {
            this.handleReset();
            $('#modalBusinessProfile').modal('show');
        },
        isValidForm() {
            if (!this.phone.trim()) {
                return false;
            }

            return true;
        },
        async handleSubmit() {
            if (!this.isValidForm() || this.loading) {
                return;
            }

            try {
                await this.submitApi();
                showSuccessInfo("Perfil comercial obtido")
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                let response = await window.http.get(`/user/business-profile?phone=${this.phone_id}`)
                const results = response.data.results;
                this.jid = results.jid;
                this.email = results.email;
                this.address = results.address;
                this.categories = results.categories || [];
                this.profileOptions = results.profile_options || {};
                this.businessHoursTimeZone = results.business_hours_timezone;
                this.businessHours = results.business_hours || [];
            } catch (error) {
                if (error.response) {
                    const message = error.response.data.message;
                    if (message.includes('not be a business account')) {
                        throw new Error('Este número não é uma conta comercial do WhatsApp ou não possui um perfil comercial público.');
                    } else if (message.includes('profile data is corrupted')) {
                        throw new Error('Os dados do perfil comercial parecem estar corrompidos. Por favor, tente novamente mais tarde.');
                    } else {
                        throw new Error(message);
                    }
                } else {
                    throw new Error('Falha ao buscar o perfil comercial. Por favor, verifique o número de telefone e tente novamente.');
                }
            } finally {
                this.loading = false;
            }
        },
        handleReset() {
            this.phone = '';
            this.jid = null;
            this.email = null;
            this.address = null;
            this.categories = [];
            this.profileOptions = {};
            this.businessHoursTimeZone = null;
            this.businessHours = [];
            this.type = window.TYPEUSER;
        },
        translateDay(day) {
            return this.dayNamesMap[day.toUpperCase()] || day;
        },
        formatBusinessHours(hours) {
            if (!hours || hours.length === 0) return 'Não disponível';
            return hours.map(h => `${this.translateDay(h.day_of_week)}: ${h.open_time} - ${h.close_time} (${h.mode})`).join(', ');
        }
    },
    template: `
    <div class="olive card" @click="openModal" style="cursor: pointer;">
        <div class="content">
        <a class="ui olive right ribbon label">Conta</a>
            <div class="header">Perfil Comercial</div>
            <div class="description">
                Obtenha informações detalhadas do perfil comercial
            </div>
        </div>
    </div>
    
    
    <!--  Modal Business Profile  -->
    <div class="ui large modal" id="modalBusinessProfile">
        <i class="close icon"></i>
        <div class="header">
            Informações do Perfil Comercial
        </div>
        <div class="content">
            <form class="ui form">
                <div class="ui info message">
                    <div class="header">
                        <i class="info circle icon"></i>
                        Informações do Perfil Comercial
                    </div>
                    <p>Este recurso funciona apenas com contas comerciais do WhatsApp que possuem um perfil comercial público configurado.</p>
                </div>
                
                <FormRecipient v-model:type="type" v-model:phone="phone"/>

                <button type="button" class="ui primary button" :class="{'loading': loading, 'disabled': !this.isValidForm() || this.loading}"
                        @click.prevent="handleSubmit">
                    Obter Perfil Comercial
                </button>
            </form>

            <div v-if="jid" class="ui segment" style="margin-top: 20px;">
                <h4 class="ui header">Detalhes do Perfil Comercial</h4>
                <div class="ui list">
                    <div class="item">
                        <i class="id badge icon"></i>
                        <div class="content">
                            <div class="header">JID</div>
                            <div class="description">{{ jid }}</div>
                        </div>
                    </div>
                    <div class="item" v-if="email">
                        <i class="mail icon"></i>
                        <div class="content">
                            <div class="header">Email</div>
                            <div class="description">{{ email }}</div>
                        </div>
                    </div>
                    <div class="item" v-if="address">
                        <i class="map marker icon"></i>
                        <div class="content">
                            <div class="header">Endereço</div>
                            <div class="description">{{ address }}</div>
                        </div>
                    </div>
                    <div class="item" v-if="categories.length > 0">
                        <i class="tags icon"></i>
                        <div class="content">
                            <div class="header">Categorias</div>
                            <div class="description">
                                <div class="ui small labels">
                                    <div v-for="category in categories" :key="category.id" class="ui label">
                                        {{ category.name }}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="item" v-if="businessHoursTimeZone">
                        <i class="clock icon"></i>
                        <div class="content">
                            <div class="header">Fuso Horário</div>
                            <div class="description">{{ businessHoursTimeZone }}</div>
                        </div>
                    </div>
                    <div class="item" v-if="businessHours.length > 0">
                        <i class="calendar icon"></i>
                        <div class="content">
                            <div class="header">Horário Comercial</div>
                            <div class="description">
                                <div class="ui tiny segments">
                                    <div v-for="hours in businessHours" :key="hours.day_of_week" class="ui segment">
                                        <strong>{{ translateDay(hours.day_of_week) }}:</strong>
                                        {{ hours.open_time }} - {{ hours.close_time }} ({{ hours.mode }})
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="item" v-if="Object.keys(profileOptions).length > 0">
                        <i class="info circle icon"></i>
                        <div class="content">
                            <div class="header">Opções de Perfil</div>
                            <div class="description">
                                <div class="ui list">
                                    <div v-for="(value, key) in profileOptions" :key="key" class="item">
                                        <strong>{{ key }}:</strong> {{ value }}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    `
} 