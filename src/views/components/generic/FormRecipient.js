export default {
    name: 'FormRecipient',
    props: {
        type: {
            type: String,
            required: true
        },
        phone: {
            type: String,
            required: true
        },
        showStatus: {
            type: Boolean,
            default: false
        }
    },
    data() {
        return {
            recipientTypes: []
        };
    },
    computed: {
        phone_id() {
            return this.phone + this.type;
        },
        showPhoneInput() {
            return this.type !== window.TYPESTATUS;
        },
        filteredRecipientTypes() {
            return this.recipientTypes.filter(type => {
                if (!this.showStatus && type.value === window.TYPESTATUS) {
                    return false;
                }
                return true;
            });
        }
    },
    mounted() {
        this.recipientTypes = [
            { value: window.TYPEUSER, text: 'Mensagem Privada' },
            { value: window.TYPEGROUP, text: 'Mensagem de Grupo' },
            { value: window.TYPENEWSLETTER, text: 'Boletim Informativo' },
            { value: window.TYPESTATUS, text: 'Status' }
        ];
    },
    methods: {
        updateType(event) {
            this.$emit('update:type', event.target.value);
            if (event.target.value === window.TYPESTATUS) {
                this.$emit('update:phone', '');
            }
        },
        updatePhone(event) {
            this.$emit('update:phone', event.target.value);
        }
    },
    template: `
    <div class="field">
        <label>Tipo</label>
        <select name="type" @change="updateType" class="ui dropdown">
            <option v-for="type in filteredRecipientTypes" :value="type.value">{{ type.text }}</option>
        </select>
    </div>
    
    <div v-if="showPhoneInput" class="field">
        <label>Telefone / ID do Grupo</label>
        <input :value="phone" aria-label="wa identifier" @input="updatePhone">
        <input :value="phone_id" disabled aria-label="whatsapp_id">
    </div>
    `
}